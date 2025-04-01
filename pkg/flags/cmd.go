package flags

// This file contains all the flags used in the cmd package.

type Flag struct {
	Full  string
	Short string
}

var (
	// Parent flags
	ConfigFlag    = Flag{Full: "config"}
	ConfigDirFlag = Flag{Full: "config-dir"}

	ClusterFlag   = Flag{Full: "cluster", Short: "c"}
	NamespaceFlag = Flag{Full: "namespace", Short: "n"}
)
