/*
 * MinIO Cloud Storage, (C) 2016, 2017, 2018 MinIO, Inc.
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
	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/lifecycle"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/policy"
)

const (
	MininumRequireCapacity = 1073741824 * 2 //5G
	UserDefinedEndpoint    = "endpoint"
)

// LambdaFSObjects - Implements fs object layer.
type LambdaFSObjects struct {
	Endpoints            EndpointList
	MountPoints          map[string]ObjectLayer
	BucketMountPointsMap map[string]string
	mu                   sync.Mutex
}

func (lfs *LambdaFSObjects) buildIndex(endpoints EndpointList) error {
	lfs.mu.Lock()
	defer lfs.mu.Unlock()
	// TODO use firs endpoint as config layer, delete FsJson???

	for _, endpoint := range endpoints {
		files, err := ioutil.ReadDir(endpoint.Path)
		if err != nil {
			return fmt.Errorf("read dir err when buildIndex: '%s'", err)
		}

		for _, file := range files {
			if file.IsDir() && file.Name() != minioMetaBucket {
				fmt.Printf("index %s -> %s !\n", file.Name(), endpoint.Path)
				lfs.BucketMountPointsMap[file.Name()] = endpoint.Path
			}
		}
	}

	// use first endpoint for config
	lfs.BucketMountPointsMap[minioMetaBucket] = endpoints[0].Path
	return nil
}

func (lfs *LambdaFSObjects) availableMountPoint(ctx context.Context) (string, error) {
	// try in fixed order
	for _, endpoint := range lfs.Endpoints {
		fs := lfs.MountPoints[endpoint.Path]
		sinfo := fs.StorageInfo(ctx)
		//TODO
		// a better way?
		if sinfo.Available > MininumRequireCapacity {
			return endpoint.Path, nil
		}
	}

	return "", fmt.Errorf("no available disk for storage")
}

func (lfs *LambdaFSObjects) getMountPoint(ctx context.Context, bucket string) (fsPath string, err error) {
	var ok bool
	fsPath, ok = lfs.BucketMountPointsMap[bucket]
	if !ok {
		lfs.mu.Lock()
		defer lfs.mu.Unlock()

		fsPath, err = lfs.availableMountPoint(ctx)
		if err != nil {
			return "", err
		} else {
			lfs.BucketMountPointsMap[bucket] = fsPath
		}
	}

	return fsPath, nil
}

// NewFSObjectLayer - initialize new fs object layer.
func NewLambdaFSObjectLayer(endpoints EndpointList) (ObjectLayer, error) {
	fsObjectLayers := map[string]ObjectLayer{}
	for _, endpoint := range endpoints {
		//layer, err := NewFSObjectLayer(endpoint.Path)
		layer, err := NewFSNoFsjsonObjectLayer(endpoint.Path)
		if err != nil {
			return nil, err
		} else {
			fsObjectLayers[endpoint.Path] = layer
		}
	}

	fs := LambdaFSObjects{
		Endpoints:            endpoints,
		MountPoints:          fsObjectLayers,
		BucketMountPointsMap: map[string]string{},
	}

	if err := fs.buildIndex(endpoints); err != nil {
		return nil, err
	}

	return &fs, nil
}

func (lfs *LambdaFSObjects) StorageInfo(ctx context.Context) StorageInfo {
	var storageInfo *StorageInfo

	for _, fsLayer := range lfs.MountPoints {
		sInfo := fsLayer.StorageInfo(ctx)
		if storageInfo == nil {
			storageInfo = &sInfo
		} else {
			storageInfo.Total += sInfo.Total
			storageInfo.Used += sInfo.Used
			storageInfo.Available += sInfo.Available
		}
	}

	return *storageInfo
}

func (lfs *LambdaFSObjects) MakeBucketWithLocation(ctx context.Context, bucket string, location string) (err error) {
	fsPath, err := lfs.getMountPoint(ctx, bucket)
	if err != nil {
		return err
	}
	err = lfs.MountPoints[fsPath].MakeBucketWithLocation(ctx, bucket, location)
	return err
}

func (lfs *LambdaFSObjects) GetBucketInfo(ctx context.Context, bucket string) (bucketInfo BucketInfo, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return bucketInfo, BucketNotFound{Bucket: bucket}
	} else {
		bucketInfo, err = lfs.MountPoints[fsPath].GetBucketInfo(ctx, bucket)
		if err == nil {
			bucketInfo.Endpoint = fsPath
		}
		return bucketInfo, err
	}
}

func (lfs *LambdaFSObjects) ListBuckets(ctx context.Context) (buckets []BucketInfo, err error) {
	var bks []BucketInfo
	for _, fsLayer := range lfs.MountPoints {
		bs, err := fsLayer.ListBuckets(ctx)
		if err != nil {
			return nil, err
		} else {
			bks = append(bks, bs[:]...)
		}
	}

	for _, bucket := range bks {
		fsPath, ok := lfs.BucketMountPointsMap[bucket.Name]
		if ok {
			bucket.Endpoint = fsPath
		}
		buckets = append(buckets, bucket)
	}

	return buckets, err
}

func (lfs *LambdaFSObjects) ListLambBuckets(ctx context.Context) (buckets []LambBucketInfo, err error) {
	logger.LogIf(ctx, NotImplemented{})
	return buckets, NotImplemented{}
}

func (lfs *LambdaFSObjects) DeleteBucket(ctx context.Context, bucket string) error {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return BucketNotFound{Bucket: bucket}
	} else {
		err := lfs.MountPoints[fsPath].DeleteBucket(ctx, bucket)
		if err == nil {
			lfs.mu.Lock()
			delete(lfs.BucketMountPointsMap, bucket)
			lfs.mu.Unlock()
		}
		return err
	}
}

func (lfs *LambdaFSObjects) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return result, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].ListObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
	}
}

func (lfs *LambdaFSObjects) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return result, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].ListObjectsV2(ctx, bucket, prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
	}
}

func (lfs *LambdaFSObjects) GetObjectNInfo(ctx context.Context, bucket, object string, rs *HTTPRangeSpec, h http.Header, lockType LockType, opts ObjectOptions) (reader *GetObjectReader, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return reader, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].GetObjectNInfo(ctx, bucket, object, rs, h, lockType, opts)
	}
}

func (lfs *LambdaFSObjects) GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string, opts ObjectOptions) (err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].GetObject(ctx, bucket, object, startOffset, length, writer, etag, opts)
	}
}

func (lfs *LambdaFSObjects) GetObjectInfo(ctx context.Context, bucket, object string, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return objInfo, BucketNotFound{Bucket: bucket}
	} else {
		objInfo, err = lfs.MountPoints[fsPath].GetObjectInfo(ctx, bucket, object, opts)
		if err == nil {
			objInfo.UserDefined[UserDefinedEndpoint] = fsPath
		}
		return objInfo, err
	}
}

func (lfs *LambdaFSObjects) PutObject(ctx context.Context, bucket, object string, data *PutObjReader, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	fsPath, err := lfs.getMountPoint(ctx, bucket)
	if err != nil {
		return objInfo, err
	}

	objInfo, err = lfs.MountPoints[fsPath].PutObject(ctx, bucket, object, data, opts)
	if err == nil {
		objInfo.UserDefined[UserDefinedEndpoint] = fsPath
	}
	return objInfo, err
}

func (lfs *LambdaFSObjects) CopyObject(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo, srcOpts, dstOpts ObjectOptions) (objInfo ObjectInfo, err error) {
	// TODO, use which bucket
	fsPath, ok := lfs.BucketMountPointsMap[srcBucket]
	if !ok {
		return objInfo, BucketNotFound{Bucket: srcBucket}
	} else {
		return lfs.MountPoints[fsPath].CopyObject(ctx, srcBucket, srcObject, destBucket, destObject, srcInfo, srcOpts, dstOpts)
	}
}

func (lfs *LambdaFSObjects) DeleteObject(ctx context.Context, bucket, object string) error {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].DeleteObject(ctx, bucket, object)
	}
}

func (lfs *LambdaFSObjects) DeleteObjects(ctx context.Context, bucket string, objects []string) (_errs []error, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return _errs, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].DeleteObjects(ctx, bucket, objects)
	}
}

func (lfs *LambdaFSObjects) ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return result, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].ListMultipartUploads(ctx, bucket, prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
	}
}

func (lfs *LambdaFSObjects) NewMultipartUpload(ctx context.Context, bucket, object string, opts ObjectOptions) (uploadID string, err error) {
	fsPath, err := lfs.getMountPoint(ctx, bucket)
	if err != nil {
		return uploadID, err
	}

	return lfs.MountPoints[fsPath].NewMultipartUpload(ctx, bucket, object, opts)
}

func (lfs *LambdaFSObjects) CopyObjectPart(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int,
	startOffset int64, length int64, srcInfo ObjectInfo, srcOpts, dstOpts ObjectOptions) (info PartInfo, err error) {
	// TODO which bucket
	fsPath, ok := lfs.BucketMountPointsMap[srcBucket]
	if !ok {
		return info, BucketNotFound{Bucket: srcBucket}
	} else {
		return lfs.MountPoints[fsPath].CopyObjectPart(ctx, srcBucket, srcObject, destBucket, destObject, uploadID, partID,
			startOffset, length, srcInfo, srcOpts, dstOpts)
	}
}

func (lfs *LambdaFSObjects) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *PutObjReader, opts ObjectOptions) (info PartInfo, err error) {
	fsPath, err := lfs.getMountPoint(ctx, bucket)
	if err != nil {
		return info, err
	}

	return lfs.MountPoints[fsPath].PutObjectPart(ctx, bucket, object, uploadID, partID, data, opts)
}

func (lfs *LambdaFSObjects) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int, opts ObjectOptions) (result ListPartsInfo, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return result, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].ListObjectParts(ctx, bucket, object, uploadID, partNumberMarker, maxParts, opts)
	}
}

func (lfs *LambdaFSObjects) AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].AbortMultipartUpload(ctx, bucket, object, uploadID)
	}
}

func (lfs *LambdaFSObjects) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []CompletePart, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	fsPath, ok := lfs.BucketMountPointsMap[bucket]
	if !ok {
		return objInfo, BucketNotFound{Bucket: bucket}
	} else {
		return lfs.MountPoints[fsPath].CompleteMultipartUpload(ctx, bucket, object, uploadID, uploadedParts, opts)
	}
}

// ReloadFormat - no-op for fs, Valid only for XL
func (lfs *LambdaFSObjects) ReloadFormat(ctx context.Context, dryRun bool) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (lfs *LambdaFSObjects) HealFormat(ctx context.Context, dryRun bool) (madmin.HealResultItem, error) {
	logger.LogIf(ctx, NotImplemented{})
	return madmin.HealResultItem{}, NotImplemented{}
}

func (lfs *LambdaFSObjects) HealBucket(ctx context.Context, bucket string, dryRun, remove bool) (madmin.HealResultItem, error) {
	logger.LogIf(ctx, NotImplemented{})
	return madmin.HealResultItem{}, NotImplemented{}
}

func (lfs *LambdaFSObjects) HealObject(ctx context.Context, bucket, object string, dryRun, remove bool, scanMode madmin.HealScanMode) (res madmin.HealResultItem, err error) {
	logger.LogIf(ctx, NotImplemented{})
	return res, NotImplemented{}
}

func (lfs *LambdaFSObjects) HealObjects(ctx context.Context, bucket, prefix string, healObjectFn func(string, string) error) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (lfs *LambdaFSObjects) ListBucketsHeal(ctx context.Context) (buckets []BucketInfo, err error) {
	logger.LogIf(ctx, NotImplemented{})
	return []BucketInfo{}, NotImplemented{}
}

func (lfs *LambdaFSObjects) ListObjectsHeal(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	logger.LogIf(ctx, NotImplemented{})
	return ListObjectsInfo{}, NotImplemented{}
}

func (lfs *LambdaFSObjects) SetBucketPolicy(ctx context.Context, bucket string, policy *policy.Policy) error {
	// only need to be called by first endpoint
	err := savePolicyConfig(ctx, lfs, bucket, policy)
	return err
}

func (lfs *LambdaFSObjects) GetBucketPolicy(ctx context.Context, bucket string) (policy *policy.Policy, err error) {
	// only need to be called by first endpoint
	policy, err = getPolicyConfig(lfs, bucket)
	return policy, err
}

func (lfs *LambdaFSObjects) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	// only need to be called by first endpoint
	return removePolicyConfig(ctx, lfs, bucket)
}

func (lfs *LambdaFSObjects) IsNotificationSupported() bool {
	return true
}

func (lfs *LambdaFSObjects) IsListenBucketSupported() bool {
	return true
}

func (lfs *LambdaFSObjects) IsEncryptionSupported() bool {
	return true
}

func (lfs *LambdaFSObjects) IsCompressionSupported() bool {
	return true
}

func (lfs *LambdaFSObjects) SetBucketLifecycle(ctx context.Context, bucket string, lifecycle *lifecycle.Lifecycle) error {
	// only need to be called by first endpoint
	return saveLifecycleConfig(ctx, lfs, bucket, lifecycle)
}

func (lfs *LambdaFSObjects) GetBucketLifecycle(ctx context.Context, bucket string) (lifecycle *lifecycle.Lifecycle, err error) {
	// only need to be called by first endpoint
	return getLifecycleConfig(lfs, bucket)
}

func (lfs *LambdaFSObjects) DeleteBucketLifecycle(ctx context.Context, bucket string) error {
	// only need to be called by first endpoint
	return removeLifecycleConfig(ctx, lfs, bucket)
}

// Shutdown - should be called when process shuts down.
func (lfs *LambdaFSObjects) Shutdown(ctx context.Context) error {
	for _, fsLayer := range lfs.MountPoints {
		err := fsLayer.Shutdown(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
