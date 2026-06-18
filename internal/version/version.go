package version

// Version is the psy CLI version. Overridden at build time via:
//   go build -ldflags "-X github.com/psyduck/psyduck/internal/version.Version=vX.Y.Z"
var Version = "0.0.0-dev"
