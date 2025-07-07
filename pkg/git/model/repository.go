package model

// ============
// == VENDOR ==
// ============

// The Vendor struct of a [Repository] (a.k.a., the organization)
type Vendor struct {
	name         string
	repositories []Repository
}

// Init initializes the [Vendor] struct
func (v *Vendor) Init(name string, repos []Repository) {
	v.name = name
	v.repositories = repos
}

// GetName returns the name of the [Vendor] struct
func (v *Vendor) GetName() string {
	return v.name
}

// GetRepositories returns the slice of [Repository] structs in the [Vendor] struct
func (v *Vendor) GetRepositories() []Repository {
	return v.repositories
}

// ================
// == REPOSITORY ==
// ================

// The Repository struct containing a slice of [File] structs
type Repository struct {
	name  string
	url   string
	files []File
}

// Init initializes the [Repository] struct
func (r *Repository) Init(name string, url string, files []File) {
	r.name = name
	r.url = url
	r.files = files
}

// GetName returns the name of the [Repository] struct
func (r *Repository) GetName() string {
	return r.name
}

// GetUrl returns the url of the [Repository] struct
func (r *Repository) GetUrl() string {
	return r.url
}

// GetFiles returns the slice of [File] structs in the [Repository] struct
func (r *Repository) GetFiles() []File {
	return r.files
}
