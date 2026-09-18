package clihelp

import "fmt"

// The version numbers stamped into everything this library generates.
//
// They were declared in four files beside the code that happened to use them,
// which made them look like private details of those files. They are not: each
// one is a promise to something already installed on a user's machine — a
// script on disk, a dispatcher another program installed, a page man(1) reads —
// and raising one is what makes that thing be replaced. Reading them together
// is the only way to see which of those promises a change breaks.
//
// Raise a number when what it stamps changes shape. Never lower one, and never
// reuse a value: the marker is compared for equality with what is on disk.

// completionScriptVersion marks the generated scripts. It is raised whenever a
// template changes in a way that already-installed scripts must pick up, so that
// the auto-install path rewrites them instead of leaving an old script in place.
const completionScriptVersion = 5

// keyDispatcherVersion is raised whenever the shared dispatcher changes. A
// snippet installs its dispatcher only when nothing newer is already in place,
// so two programs shipping different clihelp versions cannot fight over the key:
// the newer dispatcher wins, and it serves every program in the shared registry.
const keyDispatcherVersion = 5

// explainProtocolVersion is what a program tells the shared dispatcher it
// speaks, through its registry entry. It is not the dispatcher's own version:
// the dispatcher is whatever the newest installed program shipped, while this
// says how to talk to *this* program. They change independently, which is the
// whole reason the registry carries it.
const explainProtocolVersion = 1

// manPageVersion marks a generated page, so that an upgrade can tell one it
// wrote from one it did not.
const manPageVersion = 3

// integrationVersion identifies what the generated integration file contains. It
// is derived from the two template versions rather than maintained separately,
// so that raising either one invalidates every installed file.
func integrationVersion() string {
	return fmt.Sprintf("%d.%d", completionScriptVersion, keyDispatcherVersion)
}
