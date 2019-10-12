/*
 * MinIO Cloud Storage, (C) 2017, 2018 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/minio/cli"
	xhttp "github.com/minio/minio/cmd/http"
	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/certs"
)

func init() {
	logger.Init(GOPATH, GOROOT)
	logger.RegisterUIError(fmtError)
}

var (
	iamGatewayCmd = cli.Command{
		Name:            "iam-gateway",
		Usage:           "start object storage gateway with iam enabled",
		Flags:           append(ServerFlags, GlobalFlags...),
		HideHelpCommand: true,
	}
)

// RegisterIamGatewayCommand registers a new command for gateway.
func RegisterIamGatewayCommand(cmd cli.Command) error {
	cmd.Flags = append(append(cmd.Flags, ServerFlags...), GlobalFlags...)
	iamGatewayCmd.Subcommands = append(iamGatewayCmd.Subcommands, cmd)
	return nil
}

// StartIamGateway - handler for 'minio iam-gateway <name>'.
func StartIamGateway(ctx *cli.Context, gw Gateway) {
	if gw == nil {
		logger.FatalIf(errUnexpected, "Gateway implementation not initialized")
	}

	// Disable logging until gateway initialization is complete, any
	// error during initialization will be shown as a fatal message
	logger.Disable = true

	// Validate if we have access, secret set through environment.
	gatewayName := gw.Name()
	if ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, gatewayName, 1)
	}

	// Handle common command args.
	handleCommonCmdArgs(ctx)

	// Get port to listen on from gateway address
	globalMinioHost, globalMinioPort = mustSplitHostPort(globalCLIContext.Addr)

	// On macOS, if a process already listens on LOCALIPADDR:PORT, net.Listen() falls back
	// to IPv6 address ie minio will start listening on IPv6 address whereas another
	// (non-)minio process is listening on IPv4 of given port.
	// To avoid this error situation we check for port availability.
	logger.FatalIf(checkPortAvailability(globalMinioHost, globalMinioPort), "Unable to start the gateway")

	// Check and load TLS certificates.
	var err error
	globalPublicCerts, globalTLSCerts, globalIsSSL, err = getTLSConfig()
	fmt.Println("-------is ssl---", globalIsSSL)
	logger.FatalIf(err, "Invalid TLS certificate file")

	// Check and load Root CAs.
	globalRootCAs, err = getRootCAs(globalCertsCADir.Get())
	logger.FatalIf(err, "Failed to read root CAs (%v)", err)

	// Handle common env vars.
	handleCommonEnvVars()

	// Handle gateway specific env
	handleGatewayEnvVars()

	// Validate if we have access, secret set through environment.
	if !globalIsEnvCreds {
		logger.Fatal(uiErrEnvCredentialsMissingGateway(nil), "Unable to start gateway")
	}

	// Set system resources to maximum.
	logger.LogIf(context.Background(), setMaxResources())

	initNSLock(false) // Enable local namespace lock.

	// Set when gateway is enabled
	globalIsGateway = true

	router := mux.NewRouter().SkipClean(true)

	if globalEtcdClient != nil {
		// Enable STS router if etcd is enabled.
		registerSTSRouter(router)
	}

	enableConfigOps := gatewayName == "nas"

	// TODO disable admin api
	// Enable IAM admin APIs, if not just enable basic
	// operations such as profiling, server info etc.
	registerAdminRouter(router, true, true)

	// Add healthcheck router
	registerHealthCheckRouter(router)

	// Add server metrics router
	registerMetricsRouter(router)

	// Register web router when its enabled.
	if globalIsBrowserEnabled {
		logger.FatalIf(registerWebRouter(router), "Unable to configure web browser")
	}

	// Currently only NAS and S3 gateway support encryption headers.
	encryptionEnabled := gatewayName == "s3" || gatewayName == "nas"
	allowSSEKMS := gatewayName == "s3" // Only S3 can support SSE-KMS (as pass-through)

	// Add API router.
	registerAPIRouter(router, encryptionEnabled, allowSSEKMS)

	var getCert certs.GetCertificateFunc
	if globalTLSCerts != nil {
		getCert = globalTLSCerts.GetCertificate
	}

	globalHTTPServer = xhttp.NewServer([]string{globalCLIContext.Addr}, criticalErrorHandler{registerHandlers(router, globalHandlers...)}, getCert)
	globalHTTPServer.UpdateBytesReadFunc = globalConnStats.incInputBytes
	globalHTTPServer.UpdateBytesWrittenFunc = globalConnStats.incOutputBytes
	go func() {
		globalHTTPServerErrorCh <- globalHTTPServer.Start()
	}()

	signal.Notify(globalOSSignalCh, os.Interrupt, syscall.SIGTERM)

	// !!! Do not move this block !!!
	// For all gateways, the config needs to be loaded from env
	// prior to initializing the gateway layer
	{
		// Initialize server config.
		srvCfg := newServerConfig()

		// Override any values from ENVs.
		srvCfg.loadFromEnvs()

		// Load values to cached global values.
		srvCfg.loadToCachedConfigs()

		// hold the mutex lock before a new config is assigned.
		globalServerConfigMu.Lock()
		globalServerConfig = srvCfg
		globalServerConfigMu.Unlock()
	}

	newObject, err := gw.NewGatewayLayer(globalServerConfig.GetCredential())
	if err != nil {
		// Stop watching for any certificate changes.
		globalTLSCerts.Stop()

		globalHTTPServer.Shutdown()
		logger.FatalIf(err, "Unable to initialize gateway backend")
	}

	// Populate existing buckets to the etcd backend
	if globalDNSConfig != nil {
		initFederatorBackend(newObject)
	}

	if enableConfigOps {
		// Create a new config system.
		globalConfigSys = NewConfigSys()

		// Load globalServerConfig from etcd
		logger.LogIf(context.Background(), globalConfigSys.Init(newObject))

		// Start watching disk for reloading config, this
		// is only enabled for "NAS" gateway.
		globalConfigSys.WatchConfigNASDisk(newObject)
	}

	// Load logger subsystem
	loadLoggers()

	// This is only to uniquely identify each gateway deployments.
	globalDeploymentID = os.Getenv("MINIO_GATEWAY_DEPLOYMENT_ID")
	logger.SetDeploymentID(globalDeploymentID)

	var cacheConfig = globalServerConfig.GetCacheConfig()
	if len(cacheConfig.Drives) > 0 {
		var err error
		// initialize the new disk cache objects.
		globalCacheObjectAPI, err = newServerCacheObjects(cacheConfig)
		logger.FatalIf(err, "Unable to initialize disk caching")
	}

	// Re-enable logging
	logger.Disable = false

	// TODO config this path from outside
	// (yaiba) monkey hack, iam sys need local file operation
	localFS, _ := NewFSObjectLayer("/tmp/minio-gw/")
	// Create new IAM system.
	globalIAMSys = NewIAMSys()
	if err = globalIAMSys.Init(localFS); err != nil {
		logger.Fatal(err, "Unable to initialize IAM system")
	}

	// Create new policy system.
	globalPolicySys = NewPolicySys()
	// Initialize policy system.
	go globalPolicySys.Init(localFS)

	globalIgnorePolicyCheck = true
	// Create new lifecycle system
	globalLifecycleSys = NewLifecycleSys()

	// Create new notification system.
	globalNotificationSys = NewNotificationSys(globalServerConfig, globalEndpoints)
	if globalEtcdClient != nil && newObject.IsNotificationSupported() {
		logger.LogIf(context.Background(), globalNotificationSys.Init(newObject))
	}

	// Encryption support checks in gateway mode.
	{
		if (globalAutoEncryption || GlobalKMS != nil) && !newObject.IsEncryptionSupported() {
			logger.Fatal(errInvalidArgument,
				"Encryption support is requested but (%s) gateway does not support encryption", gw.Name())
		}

		if GlobalGatewaySSE.IsSet() && GlobalKMS == nil {
			logger.Fatal(uiErrInvalidGWSSEEnvValue(nil).Msg("MINIO_GATEWAY_SSE set but KMS is not configured"),
				"Unable to start gateway with SSE")
		}
	}

	// Once endpoints are finalized, initialize the new iam object api.
	globalIamObjLayerMutex.Lock()
	globalIamObjectAPI = localFS
	globalIamObjLayerMutex.Unlock()

	// Once endpoints are finalized, initialize the new object api.
	globalObjLayerMutex.Lock()
	globalObjectAPI = newObject
	globalObjLayerMutex.Unlock()

	// Prints the formatted startup message once object layer is initialized.
	if !globalCLIContext.Quiet {
		mode := globalMinioModeGatewayPrefix + gatewayName
		// Check update mode.
		checkUpdate(mode)

		// Print a warning message if gateway is not ready for production before the startup banner.
		if !gw.Production() {
			logger.StartupMessage(colorYellow("               *** Warning: Not Ready for Production ***"))
		}

		// Print gateway startup message.
		printGatewayStartupMessage(getAPIEndpoints(), gatewayName)
	}

	// Set uptime time after object layer has initialized.
	globalBootTime = UTCNow()

	handleSignals()
}

func GetIamSys() *IAMSys {
	return globalIAMSys
}