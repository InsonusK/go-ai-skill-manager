@type/service
Feature: The skill detectors together: one skill per place, each checked

 # Notation as in agent_skill.feature. All three skill detectors are
 # registered (detectors.Skills), as in the real run.

 @category/happy
 Scenario Outline: Every format is found, from the start path down
  Given a source tree
   """
   {"a/guide/SKILL.md"  :"skill:guide",
    "a/guide/docs/x.md" :"x",
    "b.skill.md"        :"skill:flat",
    "h.skill/h.skill.md":"skill:human",
    "x/notes.md"        :"n"}
   """
  When I scan "<start>" with the skill detectors
  Then the scan finds "<found>"
  And the scan issues are ""
  Examples:
   | start            | found                                                              |
   | .                | guide@a/guide:agent-dir,flat@b.skill.md:flat,human@h.skill:human-dir |
   | a                | guide@a/guide:agent-dir                                            |
   | a/guide          | guide@a/guide:agent-dir                                            |
   | h.skill          | human@h.skill:human-dir                                            |
   | b.skill.md       | flat@b.skill.md:flat                                               |
   | a/guide/docs     |                                                                    |
   | a/guide/SKILL.md |                                                                    |
   | x                |                                                                    |
   | missing          |                                                                    |

 @category/happy
 Scenario: A flat skill next to a skill's folder is a skill of its own
  Given a source tree
   """
   {"some.skill/SKILL.md":"skill:dir","some.skill.md":"skill:flat"}
   """
  When I scan "." with the skill detectors
  Then the scan finds "dir@some.skill:agent-dir,flat@some.skill.md:flat"
  And the scan issues are ""

 @category/happy
 Scenario: The marker of a human-dir skill is not a flat skill too
  Given a source tree
   """
   {"h.skill/h.skill.md":"skill:one"}
   """
  When I scan "." with the skill detectors
  Then the scan finds "one@h.skill:human-dir"
  And the scan issues are ""

 @category/negative
 Scenario: A folder two detectors select is a pattern-conflict, and no skill
  Given a source tree
   """
   {"a.skill/SKILL.md":"skill:one","a.skill/a.skill.md":"skill:two"}
   """
  When I scan "." with the skill detectors
  Then the scan finds ""
  And the scan issue 1 is
   """
   {"code":"pattern-conflict","source":"local:repo","skill":"","skill_path":"","file":"a.skill",
    "message":"selected by several detectors: agent-skill, human-skill"}
   """

 # A selected folder is not searched, so FlatSkill never sees a "*.skill.md"
 # file inside a skill's folder: the folder's own detector reports it.
 @category/negative
 Scenario Outline: A "*.skill.md" file right in a skill's folder, other than its marker, is a pattern-conflict
  Given a source tree
   """
   <files>
   """
  When I scan "." with the skill detectors
  Then the scan finds ""
  And the scan issues are "<issues>"
  Examples:
   | files                                                               | issues                          |
   | {"a/SKILL.md":"skill:one","a/a.skill.md":"skill:two"}               | pattern-conflict=a.skill.md     |
   | {"h.skill/h.skill.md":"skill:one","h.skill/other.skill.md":"skill:two"} | pattern-conflict=other.skill.md |
   | {"a/SKILL.md":"skill:one","a/b/SKILL.md":"skill:two","a/c.skill.md":"skill:three"} | pattern-conflict=c.skill.md |

 @category/negative
 Scenario Outline: A skill's marker deeper in a skill's folder is a nested-skill
  Given a source tree
   """
   <files>
   """
  When I scan "." with the skill detectors
  Then the scan finds ""
  And the scan issues are "<issues>"
  Examples:
   | files                                                               | issues                       |
   | {"a/SKILL.md":"skill:one","a/b/SKILL.md":"skill:two"}               | nested-skill=b/SKILL.md      |
   | {"a/SKILL.md":"skill:one","a/docs/x.skill.md":"skill:two"}          | nested-skill=docs/x.skill.md |
   | {"h.skill/h.skill.md":"skill:one","h.skill/b/SKILL.md":"skill:two"} | nested-skill=b/SKILL.md      |

 @category/negative
 Scenario: An issue found in a skill says which skill
  Given a source tree
   """
   {"a/SKILL.md":"skill:one","a/b/SKILL.md":"skill:two"}
   """
  When I scan "." with the skill detectors
  Then the scan issue 1 is
   """
   {"code":"nested-skill","source":"local:repo","skill":"one","skill_path":"a","file":"b/SKILL.md",
    "message":"nested-skill: b/SKILL.md"}
   """

 Scenario Outline: A top-level folder excluded from checks of the skill's source is exempt from nested-skill
  Given a source tree
   """
   <files>
   """
  And folders excluded from checks of source "<source>" are "<folders>"
  When I scan "." with the skill detectors
  Then the scan finds "<found>"
  And the scan issues are "<issues>"
  @category/happy
  Examples: excluded
   | files                                                               | source | folders       | found           | issues                           |
   | {"a/SKILL.md":"skill:one","a/examples/b/SKILL.md":"skill:two"}      | repo   | examples      | one@a:agent-dir |                                  |
   | {"a/SKILL.md":"skill:one","a/examples/b/SKILL.md":"skill:two"}      | repo   | demo,examples | one@a:agent-dir |                                  |

  @category/negative
  Examples: not excluded
   | files                                                               | source | folders       | found           | issues                           |
   | {"a/SKILL.md":"skill:one","a/examples/b/SKILL.md":"skill:two"}      | repo   | demo          |                 | nested-skill=examples/b/SKILL.md |
   | {"a/SKILL.md":"skill:one","a/examples/b/SKILL.md":"skill:two"}      | other  | examples      |                 | nested-skill=examples/b/SKILL.md |
   | {"a/SKILL.md":"skill:one","a/docs/examples/SKILL.md":"skill:two"}   | repo   | examples      |                 | nested-skill=docs/examples/SKILL.md |

 @category/negative
 Scenario: Valid skills are found next to invalid ones, which are reported as issues
  Given a source tree
   """
   {"a/SKILL.md":"skill:one","b/SKILL.md":"skill:Bad Name","c/SKILL.md":"skill:three","d.skill.md":"skill:Bad Flat"}
   """
  When I scan "." with the skill detectors
  Then the scan finds "one@a:agent-dir,three@c:agent-dir"
  And the scan issues are "invalid-skill=b,invalid-name=d.skill.md"

 @category/happy
 Scenario: A found skill's list of files is already read
  Given a source tree
   """
   {"a/SKILL.md":"skill:one","a/notes.md":"hi","a/docs/x.md":"x"}
   """
  When I scan "." with the skill detectors
  Then listing the files of the found skill "one" reads nothing more

 @category/happy
 Scenario: A skill already loaded is reused, not built again
  Given a source tree
   """
   {"a/SKILL.md":"skill:one","b/SKILL.md":"skill:two"}
   """
  And the skills at "a" are already loaded
  When I scan "." with the skill detectors
  Then the scan finds "one@a:agent-dir,two@b:agent-dir"
  And the found skill "one" is the loaded one

 @category/happy
 Scenario: A scan loads nothing into the catalog
  Given a source tree
   """
   {"a/SKILL.md":"skill:one"}
   """
  When I scan "." with the skill detectors
  Then the scan finds "one@a:agent-dir"
  And the catalog has nothing loaded at "."
  And the catalog has nothing loaded at "a"
