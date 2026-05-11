package main

import (
	"context"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (f *reloadAwareBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	return (&fakeBrokerClient{}).ReadinessGet(ctx)
}

func (f *reloadAwareBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	return (&fakeBrokerClient{}).VersionInfoGet(ctx)
}

func (f *reloadAwareBrokerClient) ProductLifecyclePostureGet(ctx context.Context) (brokerapi.ProductLifecyclePostureGetResponse, error) {
	return (&fakeBrokerClient{}).ProductLifecyclePostureGet(ctx)
}

func (f *reloadAwareBrokerClient) ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	return (&fakeBrokerClient{}).ProjectSubstratePostureGet(ctx)
}

func (f *reloadAwareBrokerClient) ProjectSubstrateAdopt(ctx context.Context) (brokerapi.ProjectSubstrateAdoptResponse, error) {
	return (&fakeBrokerClient{}).ProjectSubstrateAdopt(ctx)
}

func (f *reloadAwareBrokerClient) ProjectSubstrateInitPreview(ctx context.Context) (brokerapi.ProjectSubstrateInitPreviewResponse, error) {
	return (&fakeBrokerClient{}).ProjectSubstrateInitPreview(ctx)
}

func (f *reloadAwareBrokerClient) ProjectSubstrateInitApply(ctx context.Context, expectedPreviewToken string) (brokerapi.ProjectSubstrateInitApplyResponse, error) {
	return (&fakeBrokerClient{}).ProjectSubstrateInitApply(ctx, expectedPreviewToken)
}

func (f *reloadAwareBrokerClient) ProjectSubstrateUpgradePreview(ctx context.Context) (brokerapi.ProjectSubstrateUpgradePreviewResponse, error) {
	return (&fakeBrokerClient{}).ProjectSubstrateUpgradePreview(ctx)
}

func (f *reloadAwareBrokerClient) ProjectSubstrateUpgradeApply(ctx context.Context, expectedPreviewDigest string) (brokerapi.ProjectSubstrateUpgradeApplyResponse, error) {
	return (&fakeBrokerClient{}).ProjectSubstrateUpgradeApply(ctx, expectedPreviewDigest)
}
