package flink

import (
	"context"
	"testing"

	"github.com/lyft/flinkk8soperator/pkg/apis/app/v1alpha1"
	"github.com/lyft/flinkk8soperator/pkg/controller/flink/client"
	clientMock "github.com/lyft/flinkk8soperator/pkg/controller/flink/client/mock"
	"github.com/lyft/flinkk8soperator/pkg/controller/flink/mock"
	k8mock "github.com/lyft/flinkk8soperator/pkg/controller/k8/mock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
)

const testImage = "123.xyz.com/xx:11ae1218924428faabd9b64423fa0c332efba6b2"
const testImageKey = "11ae1"

func getTestFlinkController() FlinkController {
	return FlinkController{
		flinkJobManager:  &mock.MockJobManagerController{},
		FlinkTaskManager: &mock.MockTaskManagerController{},
		k8Cluster:        &k8mock.MockK8Cluster{},
		flinkClient:      &clientMock.MockJobManagerClient{},
	}
}

func TestFlinkIsClusterReady(t *testing.T) {
	flinkControllerForTest := getTestFlinkController()
	namespaceVal := "flink"
	labelMapVal := map[string]string{
		"imageKey": testImageKey,
	}
	mockK8Cluster := flinkControllerForTest.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.IsAllPodsRunningFunc = func(ctx context.Context, namespace string, labelMap map[string]string) (bool, error) {
		assert.Equal(t, namespaceVal, namespace)
		assert.Equal(t, labelMapVal, labelMap)
		return true, nil
	}
	flinkApp := v1alpha1.FlinkApplication{}
	flinkApp.Namespace = namespaceVal
	flinkApp.Spec.Image = testImage
	result, err := flinkControllerForTest.IsClusterReady(
		context.Background(), &flinkApp,
	)
	assert.True(t, result)
	assert.Nil(t, err)
}

func TestFlinkIsClusterChangeNeeded(t *testing.T) {
	flinkControllerForTest := getTestFlinkController()
	namespaceVal := "flink"
	labelMapVal := map[string]string{
		"imageKey": testImageKey,
	}
	mockK8Cluster := flinkControllerForTest.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.GetDeploymentsWithLabelFunc = func(ctx context.Context, namespace string, labelMap map[string]string) (*v1.DeploymentList, error) {
		assert.Equal(t, namespaceVal, namespace)
		assert.Equal(t, labelMapVal, labelMap)
		return &v1.DeploymentList{
			Items: []v1.Deployment{
				{
					Status: v1.DeploymentStatus{
						Replicas: 2,
					},
				},
			},
		}, nil
	}
	flinkApp := v1alpha1.FlinkApplication{}
	flinkApp.Namespace = namespaceVal
	flinkApp.Spec.Image = testImage
	result, err := flinkControllerForTest.IsClusterChangeNeeded(
		context.Background(), &flinkApp,
	)
	assert.False(t, result)
	assert.Nil(t, err)
}

func TestFlinkIsClusterChangeNotNeeded(t *testing.T) {
	flinkControllerForTest := getTestFlinkController()
	namespaceVal := "flink"
	labelMapVal := map[string]string{
		"imageKey": testImageKey,
	}
	mockK8Cluster := flinkControllerForTest.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.GetDeploymentsWithLabelFunc = func(ctx context.Context, namespace string, labelMap map[string]string) (*v1.DeploymentList, error) {
		assert.Equal(t, namespaceVal, namespace)
		assert.Equal(t, labelMapVal, labelMap)
		return &v1.DeploymentList{}, nil
	}
	flinkApp := v1alpha1.FlinkApplication{}
	flinkApp.Namespace = namespaceVal
	flinkApp.Spec.Image = testImage
	result, err := flinkControllerForTest.IsClusterChangeNeeded(
		context.Background(), &flinkApp,
	)
	assert.True(t, result)
	assert.Nil(t, err)
}

func TestFlinkApplicationChanged(t *testing.T) {
	flinkControllerForTest := getTestFlinkController()
	namespaceVal := "flink"
	labelMapVal := map[string]string{
		"imageKey": testImageKey,
	}
	mockK8Cluster := flinkControllerForTest.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.GetDeploymentsWithLabelFunc = func(ctx context.Context, namespace string, labelMap map[string]string) (*v1.DeploymentList, error) {
		assert.Equal(t, namespaceVal, namespace)
		assert.Equal(t, labelMapVal, labelMap)
		return &v1.DeploymentList{}, nil
	}
	flinkApp := v1alpha1.FlinkApplication{}
	flinkApp.Namespace = namespaceVal
	flinkApp.Spec.Image = testImage
	result, err := flinkControllerForTest.HasApplicationChanged(
		context.Background(), &flinkApp,
	)
	assert.True(t, result)
	assert.Nil(t, err)
}

func TestFlinkApplicationNeededNeedUpdate(t *testing.T) {
	flinkControllerForTest := getTestFlinkController()
	namespaceVal := "flink"
	appName := "abc"
	numberOfTaskManagers := int32(2)
	mockK8Cluster := flinkControllerForTest.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.GetDeploymentsWithLabelFunc = func(ctx context.Context, namespace string, labelMap map[string]string) (*v1.DeploymentList, error) {
		assert.Equal(t, namespaceVal, namespace)
		if val, ok := labelMap["imageKey"]; ok {
			assert.Equal(t, testImageKey, val)
		}
		if val, ok := labelMap["App"]; ok {
			assert.Equal(t, appName, val)
		}
		deployment := v1.Deployment{
			Spec: v1.DeploymentSpec{
				Replicas: &numberOfTaskManagers,
			},
		}
		deployment.Name = appName + "-" + testImageKey + "-tm"
		return &v1.DeploymentList{
			Items: []v1.Deployment{
				deployment,
			},
		}, nil
	}
	flinkApp := v1alpha1.FlinkApplication{}
	flinkApp.Name = appName
	flinkApp.Namespace = namespaceVal
	flinkApp.Spec.Image = testImage
	flinkApp.Spec.TaskManagerConfig.TaskManagerCount = numberOfTaskManagers + 1
	result, err := flinkControllerForTest.HasApplicationChanged(
		context.Background(), &flinkApp,
	)
	assert.True(t, result)
	assert.Nil(t, err)
}

func TestFlinkApplicationParallelismChanged(t *testing.T) {
	getJobConfigInvoked := false
	flinkControllerForTest := getTestFlinkController()
	mockK8Cluster := flinkControllerForTest.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.GetDeploymentsWithLabelFunc = func(ctx context.Context, namespace string, labelMap map[string]string) (*v1.DeploymentList, error) {
		deployment := v1.Deployment{}
		return &v1.DeploymentList{
			Items: []v1.Deployment{
				deployment,
			},
		}, nil
	}
	mockFlinkClient := flinkControllerForTest.flinkClient.(*clientMock.MockJobManagerClient)
	mockFlinkClient.GetJobConfigFunc = func(ctx context.Context, serviceName, jobId string) (*client.JobConfigResponse, error) {
		getJobConfigInvoked = true
		return &client.JobConfigResponse{
			ExecutionConfig: client.JobExecutionConfig{
				Parallelism: 3,
			},
		}, nil
	}
	flinkApp := v1alpha1.FlinkApplication{}
	flinkApp.Spec.FlinkJob.Parallelism = 2
	flinkApp.Status.ActiveJobId = "job1"
	result, err := flinkControllerForTest.HasApplicationChanged(
		context.Background(), &flinkApp,
	)
	assert.True(t, getJobConfigInvoked)
	assert.True(t, result)
	assert.Nil(t, err)
}
