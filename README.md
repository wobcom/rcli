# rcli

This was superseeded by [rucli](https://github.com/wobcom/rucli).

## Introduction

This is a NETCONF-based Junos router cli for fast and easy router handling.

This is currently tested with:

- MX204

## Build

`go mod download`  
`go build -o rcli cmd/rcli/main.go`

## Usage

rcli supports not a lot of commands. This is kinda intentional.
We are currently using [Junos Ansible solution](https://www.juniper.net/documentation/us/en/software/junos-ansible/ansible/topics/concept/junos-ansible-modules-overview.html) to manage our router.
This is quite hard to set up and dependencies need to be very well maintained.

We have a lot of people using this and a lot of small tasks, which are automated or can be done my almost anyone due to hard automation.
Therefore, we want to get rid of some hand-crafted playbooks based on the Ansible solution.

### apply

`rcli apply` applies a given configuration onto the specified router. It loads the given configuration into the candidate configuration
slot and shows a diff related to the running configuration. After a manual confirmation it applies the configuration. After three more minutes, it confirms the configuration.

### check

`rcli check` loads a given configuration onto the specified router. It loads the given configuration into the candidate configuration and shows
a diff related to the running configuration. It does **not** apply the configuration.

### exec

`rcli exec` executes an arbitrary cli command on the Junos router. Junos supports different output formats, which can be specified with `-o`. `json`, `xml` and `text` are supported.
