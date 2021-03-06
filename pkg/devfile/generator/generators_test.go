package generator

import (
	"reflect"
	"testing"

	"github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	v1 "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	"github.com/devfile/api/v2/pkg/attributes"
	"github.com/devfile/library/pkg/devfile/parser"
	v2 "github.com/devfile/library/pkg/devfile/parser/data/v2"
	"github.com/devfile/library/pkg/devfile/parser/data/v2/common"
	"github.com/devfile/library/pkg/testingutil"

	corev1 "k8s.io/api/core/v1"
)

var fakeResources corev1.ResourceRequirements

func init() {
	fakeResources, _ = testingutil.FakeResourceRequirements("0.5m", "300Mi")
}

func TestGetContainers(t *testing.T) {

	containerNames := []string{"testcontainer1", "testcontainer2"}
	containerImages := []string{"image1", "image2"}
	trueMountSources := true
	falseMountSources := false

	project := v1alpha2.Project{
		ClonePath: "test-project/",
		Name:      "project0",
		ProjectSource: v1.ProjectSource{
			Git: &v1.GitProjectSource{
				GitLikeProjectSource: v1.GitLikeProjectSource{
					Remotes: map[string]string{
						"origin": "repo",
					},
				},
			},
		},
	}

	tests := []struct {
		name                  string
		containerComponents   []v1.Component
		filterOptions         common.DevfileOptions
		wantContainerName     string
		wantContainerImage    string
		wantContainerEnv      []corev1.EnvVar
		wantContainerVolMount []corev1.VolumeMount
		wantErr               bool
	}{
		{
			name: "Case 1: Container with default project root",
			containerComponents: []v1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: v1.ComponentUnion{
						Container: &v1.ContainerComponent{
							Container: v1.Container{
								Image:        containerImages[0],
								MountSources: &trueMountSources,
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[0],
			wantContainerImage: containerImages[0],
			wantContainerEnv: []corev1.EnvVar{

				{
					Name:  "PROJECTS_ROOT",
					Value: "/projects",
				},
				{
					Name:  "PROJECT_SOURCE",
					Value: "/projects/test-project",
				},
			},
			wantContainerVolMount: []corev1.VolumeMount{
				{
					Name:      "devfile-projects",
					MountPath: "/projects",
				},
			},
		},
		{
			name: "Case 2: Container with source mapping",
			containerComponents: []v1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: v1.ComponentUnion{
						Container: &v1.ContainerComponent{
							Container: v1.Container{
								Image:         containerImages[0],
								MountSources:  &trueMountSources,
								SourceMapping: "/myroot",
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[0],
			wantContainerImage: containerImages[0],
			wantContainerEnv: []corev1.EnvVar{

				{
					Name:  "PROJECTS_ROOT",
					Value: "/myroot",
				},
				{
					Name:  "PROJECT_SOURCE",
					Value: "/myroot/test-project",
				},
			},
			wantContainerVolMount: []corev1.VolumeMount{
				{
					Name:      "devfile-projects",
					MountPath: "/myroot",
				},
			},
		},
		{
			name: "Case 3: Container with no mount source",
			containerComponents: []v1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: v1.ComponentUnion{
						Container: &v1.ContainerComponent{
							Container: v1.Container{
								Image:        containerImages[0],
								MountSources: &falseMountSources,
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[0],
			wantContainerImage: containerImages[0],
		},
		{
			name: "Case 4: Filter containers",
			containerComponents: []v1.Component{
				{
					Name: containerNames[0],
					ComponentUnion: v1.ComponentUnion{
						Container: &v1.ContainerComponent{
							Container: v1.Container{
								Image:        containerImages[0],
								MountSources: &falseMountSources,
							},
						},
					},
				},
				{
					Name: containerNames[1],
					Attributes: attributes.Attributes{}.FromStringMap(map[string]string{
						"firstString": "firstStringValue",
						"thirdString": "thirdStringValue",
					}),
					ComponentUnion: v1.ComponentUnion{
						Container: &v1.ContainerComponent{
							Container: v1.Container{
								Image:        containerImages[0],
								MountSources: &falseMountSources,
							},
						},
					},
				},
			},
			wantContainerName:  containerNames[1],
			wantContainerImage: containerImages[0],
			filterOptions: common.DevfileOptions{
				Filter: map[string]interface{}{
					"firstString": "firstStringValue",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := parser.DevfileObj{
				Data: &v2.DevfileV2{
					Devfile: v1.Devfile{
						DevWorkspaceTemplateSpec: v1.DevWorkspaceTemplateSpec{
							DevWorkspaceTemplateSpecContent: v1.DevWorkspaceTemplateSpecContent{
								Components: tt.containerComponents,
								Projects: []v1alpha2.Project{
									project,
								},
							},
						},
					},
				},
			}

			containers, err := GetContainers(devObj, tt.filterOptions)
			// Unexpected error
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetContainers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Expected error and got an err
			if tt.wantErr && err != nil {
				return
			}

			for _, container := range containers {
				if container.Name != tt.wantContainerName {
					t.Errorf("TestGetContainers error: Name mismatch - got: %s, wanted: %s", container.Name, tt.wantContainerName)
				}
				if container.Image != tt.wantContainerImage {
					t.Errorf("TestGetContainers error: Image mismatch - got: %s, wanted: %s", container.Image, tt.wantContainerImage)
				}
				if len(container.Env) > 0 && !reflect.DeepEqual(container.Env, tt.wantContainerEnv) {
					t.Errorf("TestGetContainers error: Env mismatch - got: %+v, wanted: %+v", container.Env, tt.wantContainerEnv)
				}
				if len(container.VolumeMounts) > 0 && !reflect.DeepEqual(container.VolumeMounts, tt.wantContainerVolMount) {
					t.Errorf("TestGetContainers error: Vol Mount mismatch - got: %+v, wanted: %+v", container.VolumeMounts, tt.wantContainerVolMount)
				}
			}
		})
	}

}
