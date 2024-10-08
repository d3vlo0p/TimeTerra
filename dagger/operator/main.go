// A generated module for Operator functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/operator/internal/dagger"
	"errors"
	"fmt"
	"strings"
)

type Operator struct {
	PlatformVariants []*dagger.Container
	User             *string
	Password         *dagger.Secret
}

func (m *Operator) Build(platforms []dagger.Platform, src *dagger.Directory) *Operator {
	platformVariants := make([]*dagger.Container, 0, len(platforms))

	for _, platform := range platforms {
		p := strings.Split(string(platform), "/")

		ctn := dag.Container(dagger.ContainerOpts{Platform: platform}).
			Build(src, dagger.ContainerBuildOpts{
				Dockerfile: "Dockerfile.dagger",
				BuildArgs: []dagger.BuildArg{{
					Name:  "TARGETOS",
					Value: string(p[0]),
				}, {
					Name:  "TARGETARCH",
					Value: string(p[1]),
				}},
			})

		platformVariants = append(platformVariants, ctn)
	}
	m.PlatformVariants = platformVariants
	return m
}

// set registry credentials
func (m *Operator) Credentials(user *string, password *dagger.Secret) *Operator {
	m.User = user
	m.Password = password
	return m
}

func (m *Operator) Publish(ctx context.Context,
	registry string,
	img string,
) (string, error) {
	if len(m.PlatformVariants) == 0 {
		return "", errors.New("missing containers to publish")
	}

	ctn := dag.Container()
	if m.User != nil && m.Password != nil {
		ctn = ctn.WithRegistryAuth(registry, *m.User, m.Password)
	}

	imageDigest, err := ctn.Publish(ctx, fmt.Sprintf("%s/%s", registry, img), dagger.ContainerPublishOpts{
		PlatformVariants: m.PlatformVariants,
	})
	if err != nil {
		return "", err
	}

	return imageDigest, nil
}
