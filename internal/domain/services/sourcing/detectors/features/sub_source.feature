@type/service
Feature: SubSource finds the sources a source manifest lists

 # A source manifest is a file "skill-sources.yaml" or
 # "<set>.skill-sources.yaml". The detector reads it through the
 # SourceManifestReader port; the reader here is a test one: every line of
 # a manifest is one source, "source TYPE PATH", or one problem, "problem
 # CODE SETTING MESSAGE". Found sources are written "type:path".

 Scenario Outline: What the detector selects
  Given a source tree
   """
   {"<file>":"source github one"}
   """
  When I scan "." with detector "sub-source"
  Then the scan finds sources "<sources>"
  And the scan issues are ""
  @category/happy
  Examples: a source manifest
   | file                         | sources    |
   | skill-sources.yaml           | github:one |
   | team.skill-sources.yaml      | github:one |
   | d/e/skill-sources.yaml       | github:one |
   | d/a.b.skill-sources.yaml     | github:one |

  @category/boundary
  Examples: not a source manifest
   | file                         | sources    |
   | skill-sources.yml            |            |
   | myskill-sources.yaml         |            |
   | .skill-sources.yaml          |            |
   | skill-sources.yaml.bak       |            |
   | ai-skills.yaml               |            |
   | skill-sources.yaml/x.md      |            |

 @category/happy
 Scenario: The sources of every manifest are found, a manifest's sources together and in the order written
  Given a source tree
   """
   {"b/skill-sources.yaml":"source github three",
    "a/team.skill-sources.yaml":"source github one\nsource local two",
    "skill-sources.yaml":"source github four"}
   """
  When I scan "." with detector "sub-source"
  Then the scan finds sources "github:one,local:two,github:three,github:four"

 @category/happy
 Scenario: The reader is given the manifest: its source, path, content and the repository's folder
  Given a source tree
   """
   {"docs/team.skill-sources.yaml":"source github one"}
   """
  When I scan "." with detector "sub-source"
  Then the manifest reader was given
   """
   [{"source":"local:repo","path":"docs/team.skill-sources.yaml","repo_dir":"/repos/repo","content":"source github one"}]
   """

 @category/happy
 Scenario Outline: With the skill detectors, manifests are found where skills are searched: outside skills' folders
  Given a source tree
   """
   {"skill-sources.yaml"         :"source github root",
    "a/guide/SKILL.md"           :"skill:guide",
    "a/guide/skill-sources.yaml" :"source github inside-a-skill",
    "a/team.skill-sources.yaml"  :"source github team",
    "h.skill/h.skill.md"         :"skill:human",
    "h.skill/skill-sources.yaml" :"source github inside-a-human-skill",
    "x/y/skill-sources.yaml"     :"source github deep",
    ".git/skill-sources.yaml"    :"source github git"}
   """
  When I scan "<start>" with every detector
  Then the scan finds sources "<sources>"
  And the scan finds "<skills>"
  And the scan issues are ""
  Examples:
   | start                     | sources                             | skills                                            |
   | .                         | github:team,github:root,github:deep | guide@a/guide:agent-dir,human@h.skill:human-dir   |
   | a                         | github:team                         | guide@a/guide:agent-dir                           |
   | a/guide                   |                                     | guide@a/guide:agent-dir                           |
   | x                         | github:deep                         |                                                   |
   | skill-sources.yaml        | github:root                         |                                                   |

 @category/negative
 Scenario: What the reader finds wrong is an issue at the manifest, and none of its sources is found
  Given a source tree
   """
   {"a/skill-sources.yaml":"source github one\nproblem unsafe-subpath sources[1].subpath[0] leads out\nproblem invalid-tags - no setting",
    "b/skill-sources.yaml":"source github two"}
   """
  When I scan "." with detector "sub-source"
  Then the scan finds sources "github:two"
  And the scan issues are "unsafe-subpath=a/skill-sources.yaml,invalid-tags=a/skill-sources.yaml"
  And the scan issue 1 is
   """
   {"code":"unsafe-subpath","source":"local:repo","skill":"","skill_path":"","file":"a/skill-sources.yaml",
    "message":"sources[1].subpath[0]: leads out"}
   """
  And the scan issue 2 is
   """
   {"code":"invalid-tags","source":"local:repo","skill":"","skill_path":"","file":"a/skill-sources.yaml",
    "message":"no setting"}
   """

 @category/error
 Scenario: A manifest that can't be read is an issue, the reader is not asked
  Given a source tree
   """
   {"unreadable.skill-sources.yaml":"source github one","skill-sources.yaml":"source github two"}
   """
  When I scan "." with detector "sub-source"
  Then the scan finds sources "github:two"
  And the scan issue 1 is
   """
   {"code":"source-read","source":"local:repo","skill":"","skill_path":"","file":"unreadable.skill-sources.yaml",
    "message":"disk failure"}
   """
  And the manifest reader was given
   """
   [{"source":"local:repo","path":"skill-sources.yaml","repo_dir":"/repos/repo","content":"source github two"}]
   """
