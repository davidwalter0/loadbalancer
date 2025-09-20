# Copyright 2018 David Walter.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export GOSUMDB=
.PHONY: install clean build utils install-utils yaml appl get push tag tag-push info e2e-test e2e-test-basic e2e-test-health e2e-test-debug e2e-test-ci setcap
# To enable kubernetes commands a valid configuration is required

# export GOPATH=/go
export kubectl=${GOPATH}/bin/kubectl  --kubeconfig=${PWD}/cluster/auth/kubeconfig
SHELL=/bin/bash
MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
CURRENT_DIR := $(notdir $(patsubst %/,%,$(dir $(MAKEFILE_DIR))))
export DIR=$(MAKEFILE_DIR)
export APPL=$(notdir $(PWD))
export IMAGE=$(notdir $(PWD))
# Set default DOCKER_USER if not defined
ifeq ($(DOCKER_USER),)
DOCKER_USER := davidwalter0
export DOCKER_USER
endif
# extract tag from latest commit, use tag for version
export gittag=$$(git tag -l --contains $(git log --pretty="format:%h"  -n1))
export TAG=$(shell if git diff --quiet --ignore-submodules HEAD && [[ -n $(gittag) ]]; then echo $(gittag); else echo "canary"; fi)
depends:=$(shell ls -1 */*.go| grep -v test)
build_deps:=$(wildcard *.go)
target:=bin/$(APPL)

# include Makefile.defs

all: info build utils setcap

info: 
	@echo $(info)

# Build utility binaries
utils: bin
	@echo "Building utility binaries..."
	CGO_ENABLED=0 go build --tags netgo -o bin/ ./cmd/...

# Install utility binaries to ~/go/bin
install-utils: utils
	@echo "Installing utility binaries to ~/go/bin..."
	@mkdir -p ~/go/bin
	@cp bin/findinterfaces ~/go/bin/
	@cp bin/listinterfaces ~/go/bin/
	@cp bin/getbestinterface ~/go/bin/
	@cp bin/podwatch ~/go/bin/
	@cp bin/nodes ~/go/bin/
	@cp bin/pods ~/go/bin/
	@cp bin/endpointwatch ~/go/bin/
	@cp bin/servicewatch ~/go/bin/
	@cp bin/nodewatch ~/go/bin/

etags:
	etags $(depends) $(build_deps)

.dep:
	mkdir -p .dep
	touch .dep --reference=Makefile

build: info $(target)

$(target): .dep bin $(build_deps) $(depends) Makefile
	@echo $(info)
	@echo "Building via % rule for $@ from $<"
	@echo $(depends)
	@if go version|grep -q 1.4 ; then														\
	    args=" -X main.Version $${TAG} -X main.Build $$(date -u +%Y.%m.%d.%H.%M.%S.%:::z) -X main.Commit $$(git log --format=%h-%aI -n1)";	\
	else																		\
	    args=" -X main.Version=$${TAG} -X main.Build=$$(date -u +%Y.%m.%d.%H.%M.%S.%:::z) -X main.Commit=$$(git log --format=%h-%aI -n1)";	\
	fi;																		\
	CGO_ENABLED=0 go build --tags netgo -ldflags "$${args}" -o $@ $(build_deps) ;
	touch $@

# Set capabilities on the loadbalancer binary to allow binding to low ports without root
setcap: $(target)
	@echo "Setting capabilities on $(target)..."
	@if command -v setcap >/dev/null 2>&1; then \
		sudo setcap 'cap_net_admin,cap_net_raw,cap_net_bind_service=+ep' $(target) && \
		echo "Successfully set capabilities on $(target)" || \
		echo "WARNING: Failed to set capabilities on $(target). You may need to run with sudo."; \
	else \
		echo "WARNING: setcap command not found. Install libcap2-bin package."; \
	fi

install: .dep/install

.dep/install: info .dep $(target)
	cp $(target) /go/bin/
	touch $@

image: .dep/image-$(DOCKER_USER)-$(IMAGE)-latest .dep/tag-$(DOCKER_USER)-$(IMAGE)-${TAG}

.dep/image-$(DOCKER_USER)-$(IMAGE)-latest: .dep $(target)
	docker buildx build --progress=plain --tag=$(DOCKER_USER)/$(IMAGE):latest .
	touch $@ 

tag: info .dep .dep/tag-$(DOCKER_USER)-$(IMAGE)-$(TAG)
	@echo $(info)

.dep/tag-$(DOCKER_USER)-$(IMAGE)-$(TAG): .dep/image-$(DOCKER_USER)-$(IMAGE)-latest
	docker tag $(DOCKER_USER)/$(IMAGE):latest \
	$(DOCKER_USER)/$(IMAGE):$${TAG}
	touch $@ 

push: info .dep .dep/push-$(DOCKER_USER)-$(IMAGE)-latest

.dep/push-$(DOCKER_USER)-$(IMAGE)-latest: .dep/image-$(DOCKER_USER)-$(IMAGE)-latest
	docker push $(DOCKER_USER)/$(IMAGE):latest
	touch $@

tag-push: info .dep/tag-$(DOCKER_USER)-$(IMAGE)-$(TAG) .dep/tag-push-$(DOCKER_USER)-$(IMAGE)-$(TAG)

.dep/tag-push-$(DOCKER_USER)-$(IMAGE)-$(TAG): .dep/image-$(DOCKER_USER)-$(IMAGE)-latest
	docker push $(DOCKER_USER)/$(IMAGE):$(TAG)
	touch $@

yaml: info .dep .dep/yaml-$(DOCKER_USER)-$(IMAGE)-$(TAG)

.dep/yaml-$(DOCKER_USER)-$(IMAGE)-$(TAG): .dep $(wildcard examples/manifests/*yaml.tmpl)
	applytmpl < examples/manifests/loadbalancer-daemonset.yaml.tmpl > daemonset.yaml
	touch $@

delete: .dep/delete

.dep/delete: yaml
	$(kubectl) delete ds/$(APPL) || true

deploy: info .dep/deploy

.dep/deploy: .dep yaml
	$(kubectl) apply -f daemonset.yaml

get: info .dep 

.dep/get: .dep yaml
	$(kubectl) get -f daemonset.yaml

clean: .dep bin 
	@if [[ -x "$(target)" ]]; then rm -f $(target) ; fi
	@if [[ -d "bin" ]]; then rmdir bin; fi
	rm -f .dep/*

bin:
	mkdir -p bin

# Include E2E testing targets
include Makefile.e2e
