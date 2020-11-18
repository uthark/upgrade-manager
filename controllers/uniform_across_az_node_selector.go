package controllers

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	upgrademgrv1alpha1 "github.com/keikoproj/upgrade-manager/api/v1alpha1"
	"log"
)

type azNodesCountState struct {
	TotalNodes          int
	MaxUnavailableNodes int
}

type UniformAcrossAzNodeSelector struct {
	azNodeCounts map[string]*azNodesCountState
	asg          *autoscaling.Group
}

func NewUniformAcrossAzNodeSelector(asg *autoscaling.Group, ruObj *upgrademgrv1alpha1.RollingUpgrade) *UniformAcrossAzNodeSelector {

	// find total number of nodes in each AZ
	azNodeCounts := make(map[string]*azNodesCountState)
	for _, instance := range asg.Instances {
		if _, ok := azNodeCounts[*instance.AvailabilityZone]; ok {
			azNodeCounts[*instance.AvailabilityZone].TotalNodes += 1
		} else {
			azNodeCounts[*instance.AvailabilityZone] = &azNodesCountState{TotalNodes: 1}
		}
	}

	// find max unavailable for each az
	for az, azNodeCount := range azNodeCounts {
		azNodeCount.MaxUnavailableNodes = getMaxUnavailable(ruObj.Spec.Strategy, azNodeCount.TotalNodes)
		log.Printf("Max unavailable calculated for %s, AZ %s is %d", ruObj.Name, az, azNodeCount.MaxUnavailableNodes)
	}

	return &UniformAcrossAzNodeSelector{
		azNodeCounts: azNodeCounts,
		asg:          asg,
	}
}

func (s *UniformAcrossAzNodeSelector) SelectNodesForRestack(state ClusterState, limit int) []*autoscaling.Instance {
	var instances []*autoscaling.Instance

	// Fetch instances to update from each instance group
	for az, processedState := range s.azNodeCounts {
		// Collect the needed number of instances to update
		want := limit
		if want < 0 {
			want = processedState.MaxUnavailableNodes
		}
		instancesForUpdate := getNextSetOfAvailableInstancesInAz(*s.asg.AutoScalingGroupName,
			az, processedState.MaxUnavailableNodes, s.asg.Instances, state, want)
		if instancesForUpdate == nil {
			log.Printf("No instances available for update in AZ: %s for ASG %s", az, *s.asg.AutoScalingGroupName)
		} else {
			instances = append(instances, instancesForUpdate...)
		}
	}

	return instances
}
