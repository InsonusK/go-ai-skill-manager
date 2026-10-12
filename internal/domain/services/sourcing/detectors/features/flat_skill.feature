@type/service
Feature: FlatSkill finds flat skills: a "*.skill.md" file

 # Notation as in agent_skill.feature. Only this detector is registered: a
 # "*.skill.md" file that is a folder skill's marker is told apart by the
 # other detectors, see skills.feature.

 Scenario Outline: What the detector selects
  Given a source tree
   """
   <files>
   """
  When I scan "<start>" with detector "flat-skill"
  Then the scan finds "<found>"
  And the scan issues are "<issues>"
  @category/happy
  Examples: a "*.skill.md" file
   | files                                    | start        | found                            | issues                  |
   | {"one.skill.md":"skill:one"}             | .            | one@one.skill.md:flat            |                         |
   | {"one.skill.md":"skill:one"}             | one.skill.md | one@one.skill.md:flat            |                         |
   | {"f/f.skill.md":"skill:one"}             | .            | one@f/f.skill.md:flat            |                         |
   | {"o.skill/other.skill.md":"skill:one"}   | .            | one@o.skill/other.skill.md:flat  |                         |
   | {"a.skill.md":"skill:one","b.skill.md":"skill:two"} | . | one@a.skill.md:flat,two@b.skill.md:flat |               |

  @category/boundary
  Examples: not a flat skill
   | files                                    | start        | found                            | issues                  |
   | {"notes.md":"skill:one"}                 | .            |                                  |                         |
   | {"one.skill.md.bak":"skill:one"}         | .            |                                  |                         |
   | {"x.skill.md/notes.md":"x"}              | .            |                                  |                         |
   | {"a/SKILL.md":"skill:one"}               | .            |                                  |                         |

  @category/negative
  Examples: a skill that can't be built
   | files                                    | start        | found                            | issues                  |
   | {"a.skill.md":"skill:Bad Name"}          | .            |                                  | invalid-name=a.skill.md |
