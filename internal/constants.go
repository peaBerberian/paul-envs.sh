package constants

// Version of this application
// TODO: automatize
const Version = "1.0.0"

// Version the Dockerfile, compose.yaml and project files have as semver.
// It could be considered that project files have a dependency on the base
// Dockerfile + compose.yaml file. As such a new minor for base files is
// still compatible to older project files with the same major, but not
// vice-versa.
const FileVersion = "1.0.0"

// Format of the "project.info" files: the lockfiles of the various projects.
const ProjectInfoVersion = "1.0.0"
