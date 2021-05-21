// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ecsobserver

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Tags(t *testing.T) {
	t.Run("ec2", func(t *testing.T) {
		task := Task{}
		assert.Equal(t, map[string]string(nil), task.EC2Tags())
		task.EC2 = &ec2.Instance{
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("k"),
					Value: aws.String("v"),
				},
			},
		}
		assert.Equal(t, map[string]string{"k": "v"}, task.EC2Tags())
	})

	t.Run("task", func(t *testing.T) {
		task := Task{Task: &ecs.Task{}}
		assert.Equal(t, map[string]string(nil), task.TaskTags())
		task.Task.Tags = []*ecs.Tag{
			{
				Key:   aws.String("k"),
				Value: aws.String("v"),
			},
		}
		assert.Equal(t, map[string]string{"k": "v"}, task.TaskTags())
	})

	t.Run("container", func(t *testing.T) {
		task := Task{Definition: &ecs.TaskDefinition{ContainerDefinitions: []*ecs.ContainerDefinition{{}}}}
		assert.Equal(t, map[string]string(nil), task.ContainerLabels(0))
		task.Definition.ContainerDefinitions[0].DockerLabels = map[string]*string{
			"k": aws.String("v"),
		}
		assert.Equal(t, map[string]string{"k": "v"}, task.ContainerLabels(0))
	})
}

func TestTask_PrivateIP(t *testing.T) {
	t.Run("awsvpc", func(t *testing.T) {
		task := Task{
			Task: &ecs.Task{
				TaskArn:           aws.String("arn:task:t2"),
				TaskDefinitionArn: aws.String("t2"),
				Attachments: []*ecs.Attachment{
					{
						Type: aws.String("ElasticNetworkInterface"),
						Details: []*ecs.KeyValuePair{
							{
								Name:  aws.String("privateIPv4Address"),
								Value: aws.String("172.168.1.1"),
							},
						},
					},
				},
			},
			Definition: &ecs.TaskDefinition{NetworkMode: aws.String(ecs.NetworkModeAwsvpc)},
		}
		ip, err := task.PrivateIP()
		require.NoError(t, err)
		assert.Equal(t, "172.168.1.1", ip)
	})

	t.Run("not found", func(t *testing.T) {
		task := Task{
			Task:       &ecs.Task{TaskArn: aws.String("arn:task:1")},
			Definition: &ecs.TaskDefinition{},
		}
		modes := []string{"", ecs.NetworkModeBridge, ecs.NetworkModeHost, ecs.NetworkModeAwsvpc, ecs.NetworkModeNone, "not even a network mode"}
		for _, mode := range modes {
			task.Definition.NetworkMode = aws.String(mode)
			_, err := task.PrivateIP()
			assert.Error(t, err)
			assert.IsType(t, &ErrPrivateIPNotFound{}, err)
			assert.Equal(t, mode, err.(*ErrPrivateIPNotFound).NetworkMode)
			// doing contains on error message is not good, but this line increase test coverage from 93% to 98%
			// not sure how the average coverage is calculated ...
			assert.Contains(t, err.Error(), mode)
		}
	})
}

func TestTask_MappedPort(t *testing.T) {
	ec2BridgeTask := &ecs.Task{
		TaskArn: aws.String("arn:task:1"),
		Containers: []*ecs.Container{
			{
				Name: aws.String("c1"),
				NetworkBindings: []*ecs.NetworkBinding{
					{
						ContainerPort: aws.Int64(2112),
						HostPort:      aws.Int64(2345),
					},
				},
			},
		},
	}
	// Network mode is optional for ecs and it default to ec2 bridge
	t.Run("empty is ec2 bridge", func(t *testing.T) {
		task := Task{
			Task:       ec2BridgeTask,
			Definition: &ecs.TaskDefinition{NetworkMode: nil},
			EC2:        &ec2.Instance{PrivateIpAddress: aws.String("172.168.1.1")},
		}
		ip, err := task.PrivateIP()
		require.Nil(t, err)
		assert.Equal(t, "172.168.1.1", ip)
		p, err := task.MappedPort(&ecs.ContainerDefinition{Name: aws.String("c1")}, 2112)
		require.Nil(t, err)
		assert.Equal(t, int64(2345), p)
	})

	t.Run("ec2 bridge", func(t *testing.T) {
		task := Task{
			Task:       ec2BridgeTask,
			Definition: &ecs.TaskDefinition{NetworkMode: aws.String(ecs.NetworkModeBridge)},
			EC2:        &ec2.Instance{PrivateIpAddress: aws.String("172.168.1.1")},
		}
		p, err := task.MappedPort(&ecs.ContainerDefinition{Name: aws.String("c1")}, 2112)
		require.Nil(t, err)
		assert.Equal(t, int64(2345), p)
	})

	vpcTaskDef := &ecs.TaskDefinition{
		NetworkMode: aws.String(ecs.NetworkModeAwsvpc),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Name: aws.String("c1"),
				PortMappings: []*ecs.PortMapping{
					{
						ContainerPort: aws.Int64(2112),
						HostPort:      aws.Int64(2345),
					},
				},
			},
		},
	}
	t.Run("awsvpc", func(t *testing.T) {
		task := Task{
			Task:       &ecs.Task{TaskArn: aws.String("arn:task:1")},
			Definition: vpcTaskDef,
		}
		p, err := task.MappedPort(vpcTaskDef.ContainerDefinitions[0], 2112)
		require.Nil(t, err)
		assert.Equal(t, int64(2345), p)
	})

	t.Run("host", func(t *testing.T) {
		def := vpcTaskDef
		def.NetworkMode = aws.String(ecs.NetworkModeHost)
		task := Task{
			Task:       &ecs.Task{TaskArn: aws.String("arn:task:1")},
			Definition: def,
		}
		p, err := task.MappedPort(def.ContainerDefinitions[0], 2112)
		require.Nil(t, err)
		assert.Equal(t, int64(2345), p)
	})

	t.Run("not found", func(t *testing.T) {
		task := Task{
			Task:       &ecs.Task{TaskArn: aws.String("arn:task:1")},
			Definition: &ecs.TaskDefinition{},
		}
		modes := []string{"", ecs.NetworkModeBridge, ecs.NetworkModeHost, ecs.NetworkModeAwsvpc, ecs.NetworkModeNone, "not even a network mode"}
		for _, mode := range modes {
			task.Definition.NetworkMode = aws.String(mode)
			_, err := task.MappedPort(&ecs.ContainerDefinition{Name: aws.String("c11")}, 1234)
			assert.Error(t, err)
			assert.Equal(t, mode, err.(*ErrMappedPortNotFound).NetworkMode)
			assert.Contains(t, err.Error(), mode) // for coverage
		}
	})
}

func TestTask_AddMatchedContainer(t *testing.T) {
	t.Run("different container", func(t *testing.T) {
		task := Task{
			Matched: []MatchedContainer{
				{
					ContainerIndex: 0,
					Targets: []MatchedTarget{
						{
							MatcherType: MatcherTypeService,
							Port:        1,
						},
					},
				},
			},
		}

		task.AddMatchedContainer(MatchedContainer{
			ContainerIndex: 1,
			Targets: []MatchedTarget{
				{
					MatcherType: MatcherTypeDockerLabel,
					Port:        2,
				},
			},
		})
		assert.Equal(t, []MatchedContainer{
			{
				ContainerIndex: 0,
				Targets: []MatchedTarget{
					{
						MatcherType: MatcherTypeService,
						Port:        1,
					},
				},
			},
			{
				ContainerIndex: 1,
				Targets: []MatchedTarget{
					{
						MatcherType: MatcherTypeDockerLabel,
						Port:        2,
					},
				},
			},
		}, task.Matched)
	})

	t.Run("same container different metris path", func(t *testing.T) {
		task := Task{
			Matched: []MatchedContainer{
				{
					ContainerIndex: 0,
					Targets: []MatchedTarget{
						{
							MatcherType: MatcherTypeService,
							Port:        1,
						},
					},
				},
			},
		}
		task.AddMatchedContainer(MatchedContainer{
			ContainerIndex: 0,
			Targets: []MatchedTarget{
				{
					MatcherType: MatcherTypeTaskDefinition,
					Port:        1,
					MetricsPath: "/metrics2",
				},
			},
		})
		assert.Equal(t, []MatchedContainer{
			{
				ContainerIndex: 0,
				Targets: []MatchedTarget{
					{
						MatcherType: MatcherTypeService,
						Port:        1,
					},
					{
						MatcherType: MatcherTypeTaskDefinition,
						Port:        1,
						MetricsPath: "/metrics2",
					},
				},
			},
		}, task.Matched)
	})

	t.Run("same container same metrics path", func(t *testing.T) {
		task := Task{
			Matched: []MatchedContainer{
				{
					ContainerIndex: 0,
					Targets: []MatchedTarget{
						{
							MatcherType: MatcherTypeService,
							Port:        1,
						},
					},
				},
			},
		}
		task.AddMatchedContainer(MatchedContainer{
			ContainerIndex: 0,
			Targets: []MatchedTarget{
				{
					MatcherType: MatcherTypeTaskDefinition,
					Port:        1,
					MetricsPath: "",
				},
			},
		})
		assert.Equal(t, []MatchedContainer{
			{
				ContainerIndex: 0,
				Targets: []MatchedTarget{
					{
						MatcherType: MatcherTypeService,
						Port:        1,
					},
				},
			},
		}, task.Matched)
	})
}
