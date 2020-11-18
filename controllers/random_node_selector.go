package controllers

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

// RandomNodeSelector returns nodes randomly from ASG ignoring AZ.
type RandomNodeSelector struct {
	asg *autoscaling.Group
}

func NewRandomNodeSelector(asg *autoscaling.Group) *RandomNodeSelector {
	return &RandomNodeSelector{asg: asg}
}

func (s *RandomNodeSelector) SelectNodesForRestack(state ClusterState, limit int) []*autoscaling.Instance {
	return getNextAvailableInstances(*s.asg.AutoScalingGroupName, limit, s.asg.Instances, state)
}
