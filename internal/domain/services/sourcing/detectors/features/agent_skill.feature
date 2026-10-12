@type/service
Feature: AgentSkill finds agent-dir skills: a folder holding the file SKILL.md

 # In "a source tree" the content "skill:NAME" is the file of a skill named
 # NAME. Found skills are written "name@location:format", issues
 # "code=file". Only this detector is registered here; how the detectors
 # work together is in skills.feature.

 Scenario Outline: What the detector selects
  Given a source tree
   """
   <files>
   """
  When I scan "." with detector "agent-skill"
  Then the scan finds "<found>"
  And the scan issues are "<issues>"
  @category/happy
  Examples: a folder with the marker file
   | files                                            | found                   | issues          |
   | {"a/guide/SKILL.md":"skill:one"}                 | one@a/guide:agent-dir   |                 |
   | {"a/guide/SKILL.md":"skill:one","a/guide/x.md":"x"} | one@a/guide:agent-dir |                |
   | {"g.skill/SKILL.md":"skill:one"}                 | one@g.skill:agent-dir   |                 |
   | {"SKILL.md":"skill:one"}                         | one@.:agent-dir         |                 |
   | {"p/q/SKILL.md":"skill:one"}                     | one@p/q:agent-dir       |                 |
   | {"a/SKILL.md":"skill:one","b/SKILL.md":"skill:two"} | one@a:agent-dir,two@b:agent-dir |      |

  @category/boundary
  Examples: not an agent-dir skill
   | files                                            | found                   | issues          |
   | {"d/SKILL.md/x.md":"x"}                          |                         |                 |
   | {"d/skill.md":"skill:one"}                       |                         |                 |
   | {"d/notSKILL.md":"skill:one"}                    |                         |                 |
   | {"one.skill.md":"skill:one"}                     |                         |                 |
   | {"h.skill/h.skill.md":"skill:one"}               |                         |                 |
   | {"README.md":"ordinary"}                         |                         |                 |

  @category/negative
  Examples: a skill that can't be built
   | files                                            | found                   | issues          |
   | {"a/SKILL.md":"skill:Bad Name"}                  |                         | invalid-skill=a |
   | {"a/SKILL.md":"skill:Bad Name","b/SKILL.md":"skill:two"} | two@b:agent-dir | invalid-skill=a |

 @category/boundary
 Scenario Outline: The skill is the folder: a path inside it is not one
  Given a source tree
   """
   {"a/guide/SKILL.md":"skill:one","a/guide/docs/x.md":"x"}
   """
  When I scan "<start>" with detector "agent-skill"
  Then the scan finds "<found>"
  Examples:
   | start            | found                 |
   | a                | one@a/guide:agent-dir |
   | a/guide          | one@a/guide:agent-dir |
   | a/guide/SKILL.md |                       |
   | a/guide/docs     |                       |
