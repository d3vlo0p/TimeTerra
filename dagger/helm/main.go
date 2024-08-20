// A generated module for Helm functions
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
	"dagger/helm/internal/dagger"
	"errors"
	"fmt"
)

type Helm struct {
	Ctn      *dagger.Container
	Tgz      *dagger.File
	User     *string
	Password *dagger.Secret
}

func (m *Helm) Package(
	chart *dagger.Directory,
	// +optional
	appVersion *string,
	// +optional
	version *string,
) *Helm {
	args := []string{"helm", "package", "."}
	if appVersion != nil {
		args = append(args, "--app-version", *appVersion)
	}
	if version != nil {
		args = append(args, "--version", *version)
	}
	m.Ctn = dag.Container().From("alpine/helm").
		WithMountedDirectory("/chart", chart).
		WithWorkdir("/chart").
		WithExec(args).
		WithExec([]string{"sh", "-c", "mv timeterra-*.tgz timeterra.tgz"})

	m.Tgz = m.Ctn.File("./timeterra.tgz")
	return m
}

// get container after package
func (m *Helm) Container() *dagger.Container {
	return m.Ctn
}

// set helm package to push
func (m *Helm) File(path *dagger.File) *Helm {
	m.Tgz = path
	return m
}

// set registry credentials
func (m *Helm) Credentials(user *string, password *dagger.Secret) *Helm {
	m.User = user
	m.Password = password
	return m
}

func (m *Helm) Push(ctx context.Context,
	scheme string,
	registry string,
	repository string,
) (string, error) {
	if m.Tgz == nil {
		return "", errors.New("missing file to push")
	}

	ctn := dag.Container().From("alpine/helm")
	if m.User != nil && m.Password != nil {
		ctn = ctn.WithSecretVariable("REGISTRY_PASSWORD", m.Password).
			WithExec([]string{"sh", "-c", fmt.Sprintf("echo $REGISTRY_PASSWORD | helm registry login %s -u %s --password-stdin", registry, *m.User)})
	}

	return ctn.WithWorkdir("/tmp").
		WithFile("./chart.tgz", m.Tgz).
		WithExec([]string{"sh", "-c", "helm push chart.tgz " + scheme + "://" + registry + "/" + repository}).Stdout(ctx)
}
