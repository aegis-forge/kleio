package model

import "regexp"

// ===============
// == COMPONENT ==
// ===============

// The Component struct used for external dependencies
type Component struct {
	name     string
	category string
	provider string
	history  []*Version
}

// Init initializes the [Component] struct
func (c *Component) Init(name string, category string, provider string, history []*Version) {
	c.name = name
	c.category = category
	c.history = history
}

// GetName returns the name of the [Component] struct
func (c *Component) GetName() string {
	return c.name
}

// GetCategory returns the name of the [Component] struct
func (c *Component) GetCategory() string {
	return c.category
}

// GetProvider returns the provider of the [Component] struct
func (c *Component) GetProvider() string {
	return c.provider
}

// GetHistory returns the versions of the [Component] struct
func (c *Component) GetHistory() []*Version {
	return c.history
}

// GetAllUses returns the total number of component used in the workflow
func (c *Component) GetAllUses() int {
	uses := 0

	for _, version := range c.GetHistory() {
		uses += version.uses
	}

	return uses
}

// =============
// == VERSION ==
// =============

// The Version struct representing a version of the [Component] struct
type Version struct {
	uses          int
	versionString string
	versionType   string
}

// Init initializes the [Version] struct
func (v *Version) Init(versionString string) {
	v.uses = 1
	v.versionString = versionString

	majorRegex := regexp.MustCompile(`^([vV])?\d+$`)
	completeRegex := regexp.MustCompile(`^([vV])?(0|[1-9]\d*)\.?(0|[1-9]\d*)?\.?(0|[1-9]\d*)?(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	hash := regexp.MustCompile(`^(sha256:)?.{40}$`)

	if majorRegex.MatchString(versionString) {
		v.versionType = "major"
	} else if completeRegex.MatchString(versionString) {
		v.versionType = "complete"
	} else if hash.MatchString(versionString) {
		v.versionType = "hash"
	} else {
		v.versionType = "branch/tag"
	}
}

// GetVersionString returns the version string of the [Version] struct
func (v *Version) GetVersionString() string {
	return v.versionString
}

// GetVersionType returns the type of version of the [Version] struct
func (v *Version) GetVersionType() string {
	return v.versionType
}

// GetUses returns the number of uses of the [Version] struct
func (v *Version) GetUses() int {
	return v.uses
}

// AddUses adds to the number of uses of the [Version] struct
func (v *Version) AddUses(uses int) {
	v.uses += uses
}
