package controllers

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	upgrademgrv1alpha1 "github.com/keikoproj/upgrade-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestUniformAcrossAzNodeSelectorSelectNodes(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	someAsg := "some-asg"
	mockID := "some-id"
	someLaunchConfig := "some-launch-config"
	diffLaunchConfig := "different-launch-config"
	az := "az-1"
	az2 := "az-2"
	az1Instance1 := constructAutoScalingInstance(mockID+"1-"+az, diffLaunchConfig, az)
	az2Instance1 := constructAutoScalingInstance(mockID+"1-"+az2, diffLaunchConfig, az2)
	az2Instance2 := constructAutoScalingInstance(mockID+"2-"+az2, diffLaunchConfig, az2)
	mockAsg := autoscaling.Group{AutoScalingGroupName: &someAsg,
		LaunchConfigurationName: &someLaunchConfig,
		Instances: []*autoscaling.Instance{
			az1Instance1,
			az2Instance1,
			az2Instance2,
		},
	}

	ruObj := &upgrademgrv1alpha1.RollingUpgrade{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec: upgrademgrv1alpha1.RollingUpgradeSpec{
			AsgName: someAsg,
			Strategy: upgrademgrv1alpha1.UpdateStrategy{
				Type: upgrademgrv1alpha1.UniformAcrossAzUpdateStrategy,
			},
		},
	}

	clusterState := NewClusterState()
	clusterState.initializeAsg(*mockAsg.AutoScalingGroupName, mockAsg.Instances)

	nodeSelector := NewUniformAcrossAzNodeSelector(&mockAsg, ruObj)
	instances := nodeSelector.SelectNodesForRestack(clusterState, -1)

	g.Expect(len(instances)).To(gomega.Equal(2))

	// group instances by AZ
	instancesByAz := make(map[string][]*autoscaling.Instance)
	for _, instance := range instances {
		az := instance.AvailabilityZone
		if _, ok := instancesByAz[*az]; !ok {
			instancesInAz := make([]*autoscaling.Instance, 0, len(instances))
			instancesByAz[*az] = instancesInAz
		}
		instancesByAz[*az] = append(instancesByAz[*az], instance)
	}

	// assert on number of instances in each az
	g.Expect(len(instancesByAz[az])).To(gomega.Equal(1))
	g.Expect(len(instancesByAz[az2])).To(gomega.Equal(1))
}

func TestUniformAcrossAzNodeSelectorSelectNodesOneAzComplete(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	someAsg := "some-asg"
	mockID := "some-id"
	someLaunchConfig := "some-launch-config"
	diffLaunchConfig := "different-launch-config"
	az := "az-1"
	az2 := "az-2"
	az1Instance1 := constructAutoScalingInstance(mockID+"1-"+az, diffLaunchConfig, az)
	az2Instance1 := constructAutoScalingInstance(mockID+"1-"+az2, diffLaunchConfig, az2)
	az2Instance2 := constructAutoScalingInstance(mockID+"2-"+az2, diffLaunchConfig, az2)
	mockAsg := autoscaling.Group{AutoScalingGroupName: &someAsg,
		LaunchConfigurationName: &someLaunchConfig,
		Instances: []*autoscaling.Instance{
			az1Instance1,
			az2Instance1,
			az2Instance2,
		},
	}

	ruObj := &upgrademgrv1alpha1.RollingUpgrade{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec: upgrademgrv1alpha1.RollingUpgradeSpec{
			AsgName: someAsg,
			Strategy: upgrademgrv1alpha1.UpdateStrategy{
				Type: upgrademgrv1alpha1.UniformAcrossAzUpdateStrategy,
			},
		},
	}

	clusterState := NewClusterState()
	clusterState.initializeAsg(*mockAsg.AutoScalingGroupName, mockAsg.Instances)
	clusterState.markUpdateInProgress(mockID + "1-" + az)
	clusterState.markUpdateInProgress(mockID + "1-" + az2)
	clusterState.markUpdateCompleted(mockID + "1-" + az)
	clusterState.markUpdateCompleted(mockID + "1-" + az2)

	nodeSelector := NewUniformAcrossAzNodeSelector(&mockAsg, ruObj)
	instances := nodeSelector.SelectNodesForRestack(clusterState, -1)

	g.Expect(len(instances)).To(gomega.Equal(1))
	g.Expect(instances[0].AvailabilityZone).To(gomega.Equal(&az2))
}
