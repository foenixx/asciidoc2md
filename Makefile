SHELL := /bin/zsh
cur_dir := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
conv_path := /mnt/c/personal/golang/asciidoc2md/docs
src_path := /mnt/c/SynProjects/Syntellect/Tessa/Docs
markdown_path := /mnt/c/SynProjects/Syntellect/tessa_docs/docs
mkdocs_path := $(markdown_path)/../mkdocs.yml
#dbg := '--debug'

.PHONY: gen_map convert build all clean init_docs copy_docs init_docs_pre init_md_docs init_md_pre init_md_post test

all: convert

build:
	go build

convert: gen_map
	#web client limitations
	./asciidoc2md convert ./docs/web_limits/WebClientLimitations.adoc --config settings.yml --slug=web_limits --out=$(markdown_path)/web_limits --split-level=1 $(dbg)
	#installation guide
	./asciidoc2md convert ./docs/installation/InstallationGuide.adoc --config settings.yml --slug=installation --out=$(markdown_path)/installation $(dbg)
	#user guide
	./asciidoc2md convert ./docs/user/UserGuide.adoc --config settings.yml --slug=user --out=$(markdown_path)/user $(dbg)
	#admin guide
	./asciidoc2md convert ./docs/admin/AdministratorGuide.adoc --config settings.yml --slug=admin --out=$(markdown_path)/admin $(dbg)
	#developer guide
	./asciidoc2md convert ./docs/dev/ProgrammersGuide.adoc --config settings.yml --slug=dev --out=$(markdown_path)/dev $(dbg)
	#best practices
	./asciidoc2md convert ./docs/dev/BestPractices.adoc --config settings.yml --split-level=3 --slug=ex --out=$(markdown_path)/dev/ex --image-path=../images/ $(dbg)
	#beginners guide
	./asciidoc2md convert ./docs/beginners/BeginnersGuide.adoc --config settings.yml --slug=beg --out=$(markdown_path)/beginners $(dbg)
	#linux installation guide
	./asciidoc2md convert ./docs/linux_inst/LinuxInstallationGuide.adoc --config settings.yml --slug=linux_inst --out=$(markdown_path)/linux_inst $(dbg)
	#web developer guide
	./asciidoc2md convert ./docs/web_sdk/WebProgrammersGuide.adoc --config settings.yml --slug=web_sdk --out=$(markdown_path)/web_sdk $(dbg)
	#workflow guide
	./asciidoc2md convert ./docs/workflow/WorkflowGuide.adoc --config settings.yml --slug=workflow --out=$(markdown_path)/workflow $(dbg)

gen_map: build clean
	#web client limitations
	./asciidoc2md gen-map ./docs/web_limits/WebClientLimitations.adoc --config settings.yml --slug=web_limits --split-level=1 $(dbg)
	#installation guide
	./asciidoc2md gen-map ./docs/installation/InstallationGuide.adoc --config settings.yml --slug=installation --write-nav=$(mkdocs_path) $(dbg)
	#user guide
	./asciidoc2md gen-map ./docs/user/UserGuide.adoc --config settings.yml --slug=user  --write-nav=$(mkdocs_path) $(dbg)
	#admin guide
	./asciidoc2md gen-map ./docs/admin/AdministratorGuide.adoc --config settings.yml --slug=admin  --write-nav=$(mkdocs_path) $(dbg)
	#developer guide
	./asciidoc2md gen-map ./docs/dev/ProgrammersGuide.adoc --config settings.yml --slug=dev  --write-nav=$(mkdocs_path) $(dbg)
	#best practices
	./asciidoc2md gen-map ./docs/dev/BestPractices.adoc --config settings.yml --split-level=3 --slug=ex  --write-nav=$(mkdocs_path) $(dbg)
	#beginners guide
	./asciidoc2md gen-map ./docs/beginners/BeginnersGuide.adoc --config settings.yml --slug=beg  --write-nav=$(mkdocs_path) $(dbg)
	#linux installation guide
	./asciidoc2md gen-map ./docs/linux_inst/LinuxInstallationGuide.adoc --config settings.yml --slug=linux_inst   --write-nav=$(mkdocs_path) $(dbg)
	#web developer guide
	./asciidoc2md gen-map ./docs/web_sdk/WebProgrammersGuide.adoc --config settings.yml --slug=web_sdk  --write-nav=$(mkdocs_path) $(dbg)
	#workflow guide
	./asciidoc2md gen-map ./docs/workflow/WorkflowGuide.adoc --config settings.yml --slug=workflow  --write-nav=$(mkdocs_path) $(dbg)

clean:
	- rm -f *.idmap

init_docs: init_docs_pre copy_docs
init_docs: in_path=$(src_path)
init_docs: out_path=$(conv_path)

init_docs_pre:
	rm -rf docs
	mkdir docs

init_md_docs: in_path=$(src_path)
init_md_docs: out_path=$(markdown_path)
init_md_docs: init_md_pre copy_docs init_md_post


test:
	pwd && setopt EXTENDED_GLOB && export fls1=(docs/user/*.adoc) && echo $${fls1};
 	#test -n "$fls" && echo $fls


init_md_pre:
	setopt EXTENDED_GLOB; \
	files=($(markdown_path)/^index.md(N)); \
	test -n "$${files}" && rm -rf $${files}; true

init_md_post:
	files=($(markdown_path)/**/*.adoc(N)); \
 	test -n "$${files}" && rm -f $${files}; true
	mkdir $(markdown_path)/dev/ex

copy_docs:
	cp -r $(in_path)/AdministratorGuide/ $(in_path)/UserGuide/ $(in_path)/BeginnersGuide/ $(in_path)/InstallationGuide/ $(in_path)/LinuxInstallationGuide/ $(in_path)/ProgrammersGuide/ $(in_path)/WorkflowGuide/ $(in_path)/WebProgrammersGuide/ $(in_path)/WebClientLimitations/ $(out_path)
	mv $(out_path)/AdministratorGuide $(out_path)/admin
	mv $(out_path)/UserGuide $(out_path)/user
	mv $(out_path)/BeginnersGuide $(out_path)/beginners
	mv $(out_path)/InstallationGuide $(out_path)/installation
	mv $(out_path)/LinuxInstallationGuide $(out_path)/linux_inst
	mv $(out_path)/ProgrammersGuide $(out_path)/dev
	mv $(out_path)/WorkflowGuide $(out_path)/workflow
	mv $(out_path)/WebProgrammersGuide $(out_path)/web_sdk
	mv $(out_path)/WebClientLimitations $(out_path)/web_limits
