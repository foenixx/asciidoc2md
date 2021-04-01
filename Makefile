SHELL := /bin/zsh
cur_dir := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

.DEFAULT_GOAL := all
# If "all" goal or no goal at all is specified when running make: "make all".
ifeq (,$(MAKECMDGOALS))
target_all=all
else ifneq ($(findstring all,$(MAKECMDGOALS)),)
target_all=all
endif

ifneq ($(findstring debug,$(MAKECMDGOALS)),)
dbg := '--debug'
endif


# short target names allow us to run "make user" instead of "make some/path/to/UserGuide.d"
target_names=\
	user\
	admin\
	inst\
	linux_inst\
	web_limits\
	beg\
	dev\
	kb\
	web_dev\
	workflow\
	rn

artifacts_dir = idmaps
config_file = settings.yml
dest_dir = /mnt/c/SynProjects/Syntellect/tessa_docs/docs
src_dir = /mnt/c/SynProjects/Syntellect/Tessa/Docs
# Default behaviour is to write navigation info. Could be disabled by setting "target.no_nav" variable.
write_nav_flag := --write-nav=$(dest_dir)/../mkdocs.yml
# Options for every target.
# Source file name (.adoc).
user.src = $(src_dir)/UserGuide/UserGuide.adoc
# Destination file name (.d). Destination files are synthetic, and intended only for tracking dependencies change,
#  since we don't know in advance what names the output .md file will have. All output markdown files are
#  generated in the destination directory nearby .d file.
user.dest=$(dest_dir)/user/user.d
# Set this to any non-empty value (1 or true, for example) to disable writing navigation snippet into mkdocs.yml
user.no_nav=
# Set this to a numeric value to override default --split-level value. Empty value means default value: 2.
user.split_level=
# Set this to any non-empty value (1 or true, for example) to skip copying everything from source folder
#  during destination folder initialization (folder_init.xxx targets).
user.no_images=
# Set this to non-empty value to override default image path (./image)
user.images_dir=
admin.src=$(src_dir)/AdministratorGuide/AdministratorGuide.adoc
admin.dest=$(dest_dir)/admin/admin.d
inst.src=$(src_dir)/InstallationGuide/InstallationGuide.adoc
inst.dest=$(dest_dir)/install/install.d
linux_inst.src=$(src_dir)/LinuxInstallationGuide/LinuxInstallationGuide.adoc
linux_inst.dest=$(dest_dir)/linux_inst/linux_inst.d
web_limits.src=$(src_dir)/WebClientLimitations/WebClientLimitations.adoc
web_limits.dest=$(dest_dir)/web_limits/web_limits.d
web_limits.no_nav=true
web_limits.split_level=1
beg.src=$(src_dir)/BeginnersGuide/BeginnersGuide.adoc
beg.dest=$(dest_dir)/beg/beg.d
dev.src=$(src_dir)/ProgrammersGuide/ProgrammersGuide.adoc
dev.dest=$(dest_dir)/dev/dev.d
kb.src=$(src_dir)/ProgrammersGuide/BestPractices.adoc
kb.dest=$(dest_dir)/dev/kb/kb.d
kb.split_level=3
kb.no_images=true
kb.images_dir=../images/
web_dev.src=$(src_dir)/WebProgrammersGuide/WebProgrammersGuide.adoc
web_dev.dest=$(dest_dir)/web_dev/web_dev.d
workflow.src=$(src_dir)/WorkflowGuide/WorkflowGuide.adoc
workflow.dest=$(dest_dir)/workflow/workflow.d
rn.src=$(src_dir)/ReleaseNotes/ReleaseNotes.adoc
rn.dest=$(dest_dir)/rn/rn.d


.PHONY: dest_reinit build all clean wipe_dest wipe_dest_proxy all_idmaps_proxy asciidoc2md_build debug
# For every input target name it defines all required rules
# 1. Rule for building a specific target by short name: "make user"
# 2. Rule for building a ".d" file in target directory.
# 3. Rule for building a ".idmap" file in artifacts directory.
# 4. Overrides for split level and nav writing.
# 5. Rule for creating a destination folder.
# 6. Rule "xxx.init" for copying all files (except adoc) from source to destination.
# 7. Rule "xxx.clean" for clearing destination folder.
define adoc_rule =
 $(eval target_dest=$($(1).dest))
 $(eval target_src=$($(1).src))
 $(eval target_idmap_file=$(notdir $(target_src).idmap))
 $(eval target_dest_dir=$(dir $(target_dest)))
 .PHONY: $(1)
 src_files_all+=$(target_src)
 $(1): $(target_dest)
 $(target_dest): $(target_src) $(artifacts_dir)/$(target_idmap_file) | $(target_dest_dir)
 $(target_dest): slug=$(1)
ifdef $(1).images_dir
 $(target_dest): images_flag=--image-path=$($(1).images_dir)
endif
 $(artifacts_dir)/$(target_idmap_file): $(target_src) asciidoc2md
 $(artifacts_dir)/$(target_idmap_file): slug=$(1)
ifdef $(1).no_nav
 $(artifacts_dir)/$(target_idmap_file): write_nav_flag=
endif
 idmap_files_all+=$(artifacts_dir)/$(target_idmap_file)
ifdef $(1).split_level
 $(artifacts_dir)/$(target_idmap_file): split_flag=--split-level=$($(1).split_level)
 $(target_dest): split_flag=--split-level=$($(1).split_level)
endif
 dest_files_all+= $(target_dest)

 $(target_dest_dir):
	mkdir -p $$@

 .PHONY: $(1).init
 $(1).init: wipe_dest_proxy | $(target_dest_dir)
	rm -rf 	$(target_dest_dir)
	mkdir $(target_dest_dir)
ifndef $(1).no_images
	cp -r $(dir $(target_src))/* $(target_dest_dir)
	rm $(target_dest_dir)/*.adoc
endif
 folder_init_all_targets += $(1).init

 .PHONY: $(1).clean
 $(1).clean:
	-@rm $(target_dest_dir)/*.d $(target_dest_dir)/*.md
	-@rm $(artifacts_dir)/$(target_idmap_file)

endef

#$(info $(call adoc_rule,user))
#$(foreach item,$(target_names),$(info $(call adoc_rule,$(item))))
$(foreach item,$(target_names),$(eval $(call adoc_rule,$(item))))
#$(info $(dest_files_all))

# This target makes sure that building all idmaps is done before building
#  any .md files.
all: $(target_names)
ifdef target_all
 $(dest_files_all): all_idmaps_proxy
endif
all_idmaps_proxy: $(idmap_files_all)
#$(info $(idmap_files_all))

dest_reinit: $(folder_init_all_targets)

ifneq ($(findstring dest_reinit,$(MAKECMDGOALS)),)
wipe_dest_proxy: wipe_dest
endif

wipe_dest:
	@echo "clearing destination dir"
	setopt EXTENDED_GLOB; \
  	files=($(dest_dir)/^index.md(N)); \
	test -n "$${files}" && rm -rf $${files}; true


# There are no prerequisites here. They come from adoc_rule evaluation and are merged here.
%.d:
	@echo "building $@ out of $<"
	./asciidoc2md convert $< --config $(config_file) --slug=$(slug) --art=$(artifacts_dir) --out=$(dir $@) $(split_flag) $(images_flag) $(dbg)
	touch $@
%.idmap:
	@echo "IDMAP: building $@ out of $<"
	./asciidoc2md gen-map $< --config $(config_file) --slug=$(slug) --art=$(artifacts_dir) $(write_nav_flag) $(split_flag) $(dbg)
clean:
	@echo "removing *.idmap files..."
	-rm -f $(artifacts_dir)/*.idmap
	@echo "removing *.md files..."
	rm -f $(foreach item,$(dest_files_all),$(wildcard $(dir $(item))/*.md))
	@echo "removing synthetic *.d files..."
	rm -f $(dest_files_all)

asciidoc2md: asciidoc2md_build
asciidoc2md_build:
	go build
$(artifacts_dir):
	mkdir $@

apply_adoc_fixes:
	# fix invalid list markers "•  list item1"
	sed -i -E "s/^•\s+/\* /" $(workflow.src)
    # fix invalid link "<<аналогично <<PlholderF, плейсхолдеру {f:...}>>"
	sed -i -E "s/<<аналогично <<PlholderF/аналогично <<PlholderF/" $(admin.src)
	# fix invalid asciidoc syntax in RoutingGuide.adoc:
	# * `AddTaskHistoryRecordAsync(
	#            Guid? taskHistoryGroup, ...
	sed -i -E '/`AddTaskHistoryRecordAsync\(\s?$$/bx; b ; :x ; /null\)`/by ; N; bx ; :y ; s/\s{2,}/ /g ; s/\r?\n//g ' $(dir $(admin.src))RoutingGuide.adoc
	# Jinja2 template engine (enabled by using "macros" plugin treats `{#text }` as invalid comment tags and fails.
	# Replacing `{#` with '{\u2060#' fixes the problem. \u2060 is a "word joiner" symbol (non breaking and zero width).
    # Its representation in hex format is 0xe281a0 (echo -ne '\u2060' | hexdump -C).
	sed -E -i 's/\{#/{\xe2\x81\xa0#/g' $(admin.src)
	sed -i -E -e 's/<https:\/\/www\.mytessa\.ru>/https:\/\/www.mytessa.ru/'\
 		-e 's/\(c\) Syntellect/\&copy\; Syntellect/i'\
 		-e 's/vSyntellect TESSA \{version\}/Syntellect TESSA {{ tessa.version }}/' $(src_files_all)


