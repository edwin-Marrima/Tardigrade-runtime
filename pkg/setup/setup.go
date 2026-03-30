package setup

type Segment interface {
	// Provision install the portion that makes up tardigrade runtime setup
	Provision() error
	// DeProvision uninstall the portion that makes up tardigrade runtime setup
	DeProvision() error
}
