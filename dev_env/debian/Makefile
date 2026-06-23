.PHONY: setup check system user vulkan list lint

# Full provision (installs Ansible if missing)
setup:
	./bootstrap.sh

# Dry run — report what would change, alter nothing
check:
	./bootstrap.sh --check --diff

# Privileged tasks only (apt packages, hardening, NetBird) under sudo
system:
	ansible-playbook playbook.yml -K --tags privileged

# Unprivileged tasks only (user-local toolchains, no sudo)
user:
	ansible-playbook playbook.yml --skip-tags privileged

# Opt-in: install Vulkan GPU inference packages
vulkan:
	ansible-playbook playbook.yml -K --tags vulkan

# Show which tasks would run
list:
	ansible-playbook playbook.yml --list-tasks

# Syntax/sanity check
lint:
	ansible-playbook playbook.yml --syntax-check
