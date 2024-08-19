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
)

type Helm struct {
	Ctn  *dagger.Container
	file *dagger.File
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

	m.file = m.Ctn.File("./timeterra.tgz")
	return m
}

func (m *Helm) Container() *dagger.Container {
	return m.Ctn
}

func (m *Helm) Push(ctx context.Context,
	scheme string,
	registry string,
	repository string,
	// +optional
	file *dagger.File,
	// +optional
	user *string,
	// +optional
	password *dagger.Secret,
) (string, error) {
	if file == nil {
		file = m.file
	}

	ctn := dag.Container()
	if user != nil && password != nil {
		ctn = ctn.WithRegistryAuth(registry, *user, password)
	}

	return ctn.From("alpine/helm").
		WithWorkdir("/tmp").
		WithFile("./chart.tgz", file).
		WithExec([]string{"sh", "-c", "helm push chart.tgz " + scheme + "://" + registry + "/" + repository}).Stdout(ctx)
}
