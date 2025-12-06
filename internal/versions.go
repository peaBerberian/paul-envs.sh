package versions

import "github.com/peaberberian/paul-envs/internal/utils"

// Version of this application
// TODO: automatize
var Version = utils.Version{
	Major: 0,
	Minor: 1,
	Patch: 0,
}

// Version the Dockerfile, compose.yaml and project files have as semver.
// It could be considered that project files have a dependency on the base
// Dockerfile + compose.yaml file. As such a new minor for base files is
// still compatible to older project files with the same major, but not
// vice-versa.
var DockerfileVersion = utils.Version{
	Major: 1,
	Minor: 0,
	Patch: 0,
}

// Format of the "project.lock" files: the lockfiles of the various projects.
var ProjectLockVersion = utils.Version{
	Major: 1,
	Minor: 0,
	Patch: 0,
}

// Format of the "project.buildinfo" files: Information on the last build performed for a project
var BuildInfoVersion = utils.Version{
	Major: 1,
	Minor: 0,
	Patch: 0,
}
