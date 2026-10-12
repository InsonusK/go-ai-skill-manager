@type/mapping
Feature: Reading a source manifest: the sources a source names itself

 # A source manifest is the "sources" list of a configuration and nothing
 # else. Its sources are read and checked like a configuration's, so what
 # comes out is as valid as configured sources.

 @category/happy
 Scenario: Sources are read with the defaults of a configuration
  Given a source manifest "docs/skills/skill-sources.yaml" in github source "https://github.com/o/parent.git"
   """
   sources:
     - path: https://github.com/o/one.git
       type: github
     - path: https://github.com/o/two.git
       type: github
       tree: v2
       subpath: [docs/skills, extra]
       exclude_from_checks: [demo]
   """
  When I read the source manifest
  Then the manifest sources are
   """
   [{"type":"github","path":"https://github.com/o/one.git","tree":"master","subpaths":["skills"],"tags":[],"exclude":[]},
    {"type":"github","path":"https://github.com/o/two.git","tree":"v2","subpaths":["docs/skills","extra"],"tags":[],"exclude":["demo"]}]
   """

 @category/boundary
 Scenario: A manifest may list no source
  Given a source manifest "skill-sources.yaml" in github source "https://github.com/o/parent.git"
   """
   sources: []
   """
  When I read the source manifest
  Then the manifest sources are
   """
   []
   """

 @category/happy
 Scenario Outline: In a local source, a local source's path is taken from the manifest's own folder
  Given a source manifest "<manifest>" in local source "/work/repo"
   """
   sources:
     - path: <path>
   """
  When I read the source manifest
  Then the manifest sources are
   """
   [{"type":"local","path":"<resolved>","tree":"master","subpaths":[],"tags":[],"exclude":[]}]
   """
  Examples:
   | manifest                    | path        | resolved                |
   | skill-sources.yaml          | shared      | /work/repo/shared       |
   | docs/skill-sources.yaml     | shared      | /work/repo/docs/shared  |
   | docs/skill-sources.yaml     | ./shared    | /work/repo/docs/shared  |
   | docs/skill-sources.yaml     | ../shared   | /work/repo/shared       |
   | docs/a/b.skill-sources.yaml | ../../../up | /work/up                |
   | docs/skill-sources.yaml     | /abs/x/../y | /abs/y                  |

 @category/security
 Scenario: A local source in a manifest of a downloaded source is refused, and no source of the manifest is read
  Given a source manifest "docs/skill-sources.yaml" in github source "https://github.com/o/parent.git"
   """
   sources:
     - path: https://github.com/o/one.git
       type: github
     - path: /home/user/.ssh
     - path: ../../etc
       type: local
   """
  When I read the source manifest
  Then the manifest issues are
   """
   [["local-sub-source","local:/home/user/.ssh@master","sources[1].type"],
    ["local-sub-source","local:../../etc@master","sources[2].type"]]
   """
  And the manifest issue 1 says "a local source \"/home/user/.ssh\" is allowed only in a manifest of a local source, this one lies in github:https://github.com/o/parent.git@master"

 @category/negative
 Scenario Outline: A file that is not a source manifest is refused
  Given a source manifest "skill-sources.yaml" in github source "https://github.com/o/parent.git"
   """
   <content>
   """
  When I read the source manifest
  Then the manifest is refused as "invalid-manifest" with "<message>"
  Examples:
   | content                                                    | message                                                        |
   | settings: {dry_run: true}\nsources: []                     | a source manifest holds only \"sources\", found: settings      |
   | target: {default: {}}\nsettings: {}\nsources: []           | found: settings, target                                        |
   | target: {default: {}}                                      | found: target                                                  |
   | {}                                                         | a source manifest must hold \"sources\"                        |
   |                                                            | source manifest must be a mapping                              |
   | [a, b]                                                     | source manifest:                                               |
   | sources: {a: b}                                            | sources must be a list                                         |
   | sources: [text]                                            | source must be a mapping                                       |
   | sources: [{path: x, type: svn}]                            | unknown source type \"svn\"                                    |
   | sources: [{type: github}]                                  | source path is required                                        |
   | sources: [{path: o/r, type: github, name: x}]              | sources[0].name: removed setting                               |
   | sources: [{path: o/r, type: github, skip_folder: [a]}]     | sources[0].skip_folder                                         |

 @category/negative
 Scenario: The sources are checked like a configuration's, each issue naming its setting
  Given a source manifest "skill-sources.yaml" in github source "https://github.com/o/parent.git"
   """
   sources:
     - path: https://github.com/o/one.git
       type: github
       subpath: [ok, ../out]
     - path: https://github.com/o/two.git
       type: github
       tags: [go]
     - path: https://github.com/o/one.git
       type: github
       subpath: [ok, ../out]
   """
  When I read the source manifest
  Then the manifest issues are
   """
   [["unsupported-tags","github:https://github.com/o/two.git@master","sources[1].tags"],
    ["unsafe-subpath","github:https://github.com/o/one.git@master","sources[0].subpath[1]"],
    ["unsafe-subpath","github:https://github.com/o/one.git@master","sources[2].subpath[1]"],
    ["duplicate-source","github:https://github.com/o/one.git@master","sources[2]"]]
   """

 @category/security
 Scenario: A local source's subpath is checked against its resolved path
  Given a source manifest "docs/skill-sources.yaml" in local source "/work/repo"
   """
   sources:
     - path: shared
       subpath: [/work/repo/docs/shared/a, /work/repo/other]
   """
  When I read the source manifest
  Then the manifest issues are
   """
   [["unsafe-subpath","local:/work/repo/docs/shared@master","sources[0].subpath[1]"]]
   """
