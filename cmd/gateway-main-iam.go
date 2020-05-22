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
	"github.com/gorilla/mux"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	xhttp "github.com/minio/minio/cmd/http"
	"github.com/minio/minio/cmd/logger"
	"os"
	"os/signal"
	"syscall"
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
	// Only S3 can support SSE-KMS (as pass-through)
	// Add API router.
	registerAPIRouter(router, true, false)

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

	// (yaiba) monkey hack, iam sys need local file operation, make it to legacy configDir
	iamFs, err := NewFSObjectLayer(globalConfigDir.Get())
	if err != nil {
		logger.FatalIf(err, "Unable to initialize iam layer")
	}

	{
		// Create a new config system.
		globalConfigSys = NewConfigSys()

		// Load globalServerConfig from etcd
		logger.LogIf(context.Background(), globalConfigSys.Init(iamFs))

		// Start watching disk for reloading config
		globalConfigSys.WatchConfigNASDisk(iamFs)
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
		globalCacheObjectAPI, err = newServerCacheObjects(context.Background(), cacheConfig)
		logger.FatalIf(err, "Unable to initialize disk caching")
	}

	// Re-enable logging
	logger.Disable = false

	// Create new IAM system.
	globalIAMSys = NewIAMSys()
	// Initialize IAM sys.
	if err = globalIAMSys.Init(iamFs); err != nil {
		logger.Fatal(err, "Unable to initialize IAM system")
	}

	// Create new policy system.
	globalPolicySys = NewPolicySys()
	// Initialize policy system.
	if err = globalPolicySys.Init(iamFs); err != nil {
		logger.Fatal(err, "Unable to initialize policy system")
	}

	// Create new lifecycle system
	globalLifecycleSys = NewLifecycleSys()

	// Create new notification system.
	globalNotificationSys = NewNotificationSys(globalServerConfig, globalEndpoints)
	if newObject.IsNotificationSupported() {
		logger.LogIf(context.Background(), globalNotificationSys.Init(newObject))
	}

	// Verify if object layer supports
	// - encryption
	// - compression
	verifyObjectLayerFeatures("gateway "+gatewayName, newObject)

	// Once endpoints are finalized, initialize the new iam object api.
	globalIamObjLayerMutex.Lock()
	globalIamObjectAPI = iamFs
	globalIamObjLayerMutex.Unlock()

	// Once endpoints are finalized, initialize the new object api.
	globalObjLayerMutex.Lock()
	globalObjectAPI = newObject
	globalObjLayerMutex.Unlock()

	// lamb custom startup message, always print
	console.Println(colorBlue("\nGateway name:    ") + gw.Name())
	console.Println(colorBlue("TLS enabled:     ") + fmt.Sprintf("%v", globalIsSSL))
	console.Println(colorBlue("Iam config path: ") +  globalConfigDir.Get() + "\n")

	// Prints the formatted startup message once object layer is initialized.
	if !globalCLIContext.Quiet {
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
