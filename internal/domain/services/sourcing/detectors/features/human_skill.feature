@type/service
Feature: HumanSkill finds human-dir skills: a folder "x.skill" holding the file "x.skill.md"

 # Notation as in agent_skill.feature. Only this detector is registered.

 Scenario Outline: What the detector selects
  Given a source tree
   """
   <files>
   """
  When I scan "." with detector "human-skill"
  Then the scan finds "<found>"
  And the scan issues are "<issues>"
  @category/happy
  Examples: a "*.skill" folder with the marker file named after it
   | files                                                  | found                     | issues                |
   | {"h.skill/h.skill.md":"skill:one"}                     | one@h.skill:human-dir     |                       |
   | {"a/h.skill/h.skill.md":"skill:one"}                   | one@a/h.skill:human-dir   |                       |
   | {"h.skill/h.skill.md":"skill:one","h.skill/x.md":"x"}  | one@h.skill:human-dir     |                       |

  @category/boundary
  Examples: not a human-dir skill
   | files                                                  | found                     | issues                |
   | {"o.skill/other.skill.md":"skill:one"}                 |                           |                       |
   | {"f/f.skill.md":"skill:one"}                           |                           |                       |
   | {"h.skill/SKILL.md":"skill:one"}                       |                           |                       |
   | {"h.skill/h.skill.md/x.md":"x"}                        |                           |                       |
   | {"h.skill/h.md":"skill:one"}                           |                           |                       |
   | {"h.skill.md":"skill:one"}                             |                           |                       |

  @category/negative
  Examples: a skill that can't be built
   | files                                                  | found                     | issues                |
   | {"h.skill/h.skill.md":"skill:Bad Name"}                |                           | invalid-skill=h.skill |
