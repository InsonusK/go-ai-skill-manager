@type/service
Feature: SourceLoader loads the configured sources and the sources they name themselves

 # A source names further sources in source manifests ("skill-sources.yaml"
 # outside its skills' folders). Here every line of a manifest is one
 # source: "repository[@tree] [subpath,subpath] [!folder,folder]" -- the
 # "!" list is its exclude_from_checks; the configured sources are written
 # the same way, separated by ";". "skill:NAME" is the file of a skill
 # named NAME. Loaded skills are written "repository[@tree]:skill".

 @category/happy
 Scenario: Sources named by sources are loaded too, level by level, after the configured ones
  Given repository "one" holds
   """
   {"a/SKILL.md":"skill:a","skill-sources.yaml":"two\nthree"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b","deep/skill-sources.yaml":"four"}
   """
  And repository "three" holds
   """
   {"c/SKILL.md":"skill:c"}
   """
  And repository "four" holds
   """
   {"d/SKILL.md":"skill:d"}
   """
  And repository "five" holds
   """
   {"e/SKILL.md":"skill:e"}
   """
  When I load the sources "one;five"
  Then the loaded skills are "one:a,five:e,two:b,three:c,four:d"
  And the load issues are
   """
   []
   """

 @category/boundary
 Scenario: Only manifests under the selected subpaths count
  Given repository "one" holds
   """
   {"in/a/SKILL.md":"skill:a","in/skill-sources.yaml":"two","skill-sources.yaml":"three","out/skill-sources.yaml":"three"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b"}
   """
  And repository "three" holds
   """
   {"c/SKILL.md":"skill:c"}
   """
  When I load the sources "one in"
  Then the loaded skills are "one:a,two:b"

 @category/boundary
 Scenario: A listed source is loaded by its own subpaths
  Given repository "one" holds
   """
   {"skill-sources.yaml":"two b,c"}
   """
  And repository "two" holds
   """
   {"a/SKILL.md":"skill:a","b/SKILL.md":"skill:b","c/x/SKILL.md":"skill:c"}
   """
  When I load the sources "one"
  Then the loaded skills are "two:b,two:c"

 @category/boundary
 Scenario: Sources naming each other end, each loaded once
  Given repository "one" holds
   """
   {"a/SKILL.md":"skill:a","skill-sources.yaml":"two"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b","skill-sources.yaml":"one\ntwo"}
   """
  When I load the sources "one"
  Then the loaded skills are "one:a,two:b"
  And the repositories were acquired as "one,two"
  And the load issues are
   """
   []
   """

 # A search with issues is not remembered as complete by the catalog, so
 # nothing but the loader keeps such a source from being searched again.
 @category/negative
 Scenario: Sources naming each other end even when a search reports issues, each issue once
  Given repository "one" holds
   """
   {"a/SKILL.md":"skill:Bad Name","skill-sources.yaml":"two"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b","skill-sources.yaml":"one"}
   """
  When I load the sources "one"
  Then the loaded skills are "two:b"
  And the load issues are
   """
   [["invalid-skill","local:one","a"]]
   """

 @category/happy
 Scenario: A source two manifests list is loaded once
  Given repository "one" holds
   """
   {"skill-sources.yaml":"two\nthree"}
   """
  And repository "two" holds
   """
   {"skill-sources.yaml":"shared"}
   """
  And repository "three" holds
   """
   {"skill-sources.yaml":"shared"}
   """
  And repository "shared" holds
   """
   {"s/SKILL.md":"skill:Bad Name","t/SKILL.md":"skill:t"}
   """
  When I load the sources "one"
  Then the loaded skills are "shared:t"
  And the load issues are
   """
   [["invalid-skill","local:shared","s"]]
   """

 @category/happy
 Scenario Outline: A source listed again is loaded for the subpaths nobody asked for yet
  Given repository "one" holds
   """
   {"skill-sources.yaml":"two <listed>"}
   """
  And repository "two" holds
   """
   {"a/SKILL.md":"skill:a","b/SKILL.md":"skill:b","bad/SKILL.md":"skill:Bad Name"}
   """
  When I load the sources "<configured>"
  Then the loaded skills are "<skills>"
  And the load issues are
   """
   <issues>
   """
  Examples:
   | configured  | listed  | skills      | issues                                |
   | one;two a   | a,b     | two:a,two:b | []                                    |
   | one;two a   | ./a,a/  | two:a       | []                                    |
   | one;two bad | bad,b   | two:b       | [["invalid-skill","local:two","bad"]] |
   | two a;one   | b       | two:a,two:b | []                                    |

 @category/negative
 Scenario Outline: A listed source asking for another tree of a known repository is not loaded
  Given repository "one" holds
   """
   {"skill-sources.yaml":"<listed>"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b","skill-sources.yaml":"<listed by two>"}
   """
  When I load the sources "<configured>"
  Then the loaded skills are "<skills>"
  And the load issues are
   """
   <issues>
   """
  Examples: the configuration knows the repository
   | configured         | listed  | listed by two | skills           | issues                                                         |
   | one;two@master     | two@v2  |               | two@master:b     | [["conflicting-tree","local:one","skill-sources.yaml"]]        |
   | one;two@master     | two@master |            | two@master:b     | []                                                             |
   | one;two@v1;two@v2  | two@v2  |               | two@v1:b,two@v2:b | []                                                            |
   | one;two@v1;two@v2  | two@v3  |               | two@v1:b,two@v2:b | [["conflicting-tree","local:one","skill-sources.yaml"]]       |

  Examples: an earlier manifest knows the repository
   | configured         | listed  | listed by two | skills           | issues                                                         |
   | one                | two@v1  | two@v2        | two@v1:b         | [["conflicting-tree","local:two@v1","skill-sources.yaml"]]     |
   | one                | two@v1\\ntwo@v2 |        | two@v1:b         | [["conflicting-tree","local:one","skill-sources.yaml"]]        |

 @category/negative
 Scenario: A tree conflict says which repository and which tree is already loaded
  Given repository "one" holds
   """
   {"docs/team.skill-sources.yaml":"two@v2"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b"}
   """
  When I load the sources "one;two@master"
  Then the load issues are
   """
   [["conflicting-tree","local:one","docs/team.skill-sources.yaml"]]
   """
  And load issue 1 says "local:two@v2: the repository is already loaded at tree \"master\""

 @category/negative
 Scenario: A manifest with a problem is an issue, the other sources are still loaded
  Given repository "one" holds
   """
   {"a/SKILL.md":"skill:a","x/skill-sources.yaml":"two\nbroken","y/skill-sources.yaml":"three\ngone"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b"}
   """
  And repository "three" holds
   """
   {"c/SKILL.md":"skill:c"}
   """
  When I load the sources "one"
  Then the loaded skills are "one:a,three:c"
  And the load issues are
   """
   [["invalid-manifest","local:one","x/skill-sources.yaml"],
    ["source-acquire","local:gone",""]]
   """

 Scenario Outline: A listed source's folders excluded from checks are those of every source and its own, unless its repository has a list already
  Given repository "one" holds
   """
   {"skill-sources.yaml":"<listed>"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b","b/examples/x/SKILL.md":"skill:x","b/demo/y/SKILL.md":"skill:y"}
   """
  And folders excluded from checks of every source are "examples"
  And folders excluded from checks of the configured source "two@v1" are "examples,demo"
  When I load the sources "<configured>"
  Then the loaded skills are "<skills>"
  And the load issues are
   """
   <issues>
   """
  @category/happy
  Examples: excluded
   | configured | listed            | skills   | issues |
   | one        | two !demo         | two:b    | []     |
   | one        | two@v1            | two@v1:b | []     |
   | one        | two@v1 !other     | two@v1:b | []     |

  @category/negative
  Examples: not excluded
   | configured | listed            | skills   | issues                                           |
   | one        | two               |          | [["nested-skill","local:two","demo/y/SKILL.md"]] |
   | one        | two !other        |          | [["nested-skill","local:two","demo/y/SKILL.md"]] |

 @category/boundary
 Scenario: The first manifest's list of excluded folders stays for its repository
  Given repository "one" holds
   """
   {"skill-sources.yaml":"two a !demo\ntwo b"}
   """
  And repository "two" holds
   """
   {"a/SKILL.md":"skill:a","a/demo/x/SKILL.md":"skill:x","b/SKILL.md":"skill:b","b/demo/y/SKILL.md":"skill:y"}
   """
  When I load the sources "one"
  Then the loaded skills are "two:a,two:b"

 @category/boundary
 Scenario: Without a scanner reading manifests only the configured sources are loaded
  Given repository "one" holds
   """
   {"a/SKILL.md":"skill:a","skill-sources.yaml":"two"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b"}
   """
  And source manifests are not read
  When I load the sources "one"
  Then the loaded skills are "one:a"
  And the repositories were acquired as "one"

 @category/error
 Scenario: Canceled loading stops at the first source
  Given repository "one" holds
   """
   {"a/SKILL.md":"skill:a"}
   """
  And repository "two" holds
   """
   {"b/SKILL.md":"skill:b"}
   """
  And loading is canceled
  When I load the sources "one;two"
  Then the loaded skills are ""
  And the load issues are
   """
   [["canceled","local:one","."]]
   """
