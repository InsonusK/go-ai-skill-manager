@type/service
Feature: A Scanner walks a repository once for all of its detectors

 # The detectors here are test ones: detector "notes:.txt" is named "notes"
 # and selects every file or folder whose name ends with ".txt"; what it
 # finds is written as "notes=<path>". It can't build an object named "unbuildable..."
 # (its Select reports invalid-skill) and finds one named "invalid..." not
 # valid (its IsValid reports nested-skill).

 @category/happy
 Scenario Outline: Objects of every detector are found at or below the start path, in the order walked
  Given a source tree
   """
   {"a/one.txt":"1","a/b/two.md":"2","c.txt":"3","d/readme":"4"}
   """
  When I scan "<start>" with detectors "notes:.txt,docs:.md"
  Then the scan finds "<found>"
  And the scan issues are ""
  Examples:
   | start     | found                                       |
   | .         | docs=a/b/two.md,notes=a/one.txt,notes=c.txt |
   | a         | docs=a/b/two.md,notes=a/one.txt             |
   | a/b       | docs=a/b/two.md                             |
   | c.txt     | notes=c.txt                                 |
   | d         |                                             |
   | d/readme  |                                             |
   | missing   |                                             |

 @category/happy
 Scenario: A selected folder is one object and is not searched into
  Given a source tree
   """
   {"box.dir/one.txt":"1","box.dir/deep/two.txt":"2","other/three.txt":"3"}
   """
  When I scan "." with detectors "boxes:.dir,notes:.txt"
  Then the scan finds "boxes=box.dir,notes=other/three.txt"
  And the scan issues are ""
  And no directory under "box.dir/deep" was read

 @category/negative
 Scenario: An entry several detectors select is an issue, and nobody gets it
  Given a source tree
   """
   {"both.txt":"1","box.txt/inner.txt":"2","c.txt":"3"}
   """
  When I scan "." with detectors "notes:.txt,texts:.txt,inner:inner.txt"
  Then the scan finds ""
  And the scan issues are "pattern-conflict=both.txt,pattern-conflict=box.txt,pattern-conflict=c.txt"
  And the scan issue at "both.txt" says "notes, texts"

 @category/negative
 Scenario: An object that can't be built or isn't valid is an issue, the rest are still found
  Given a source tree
   """
   {"a.txt":"1","invalid.txt":"2","unbuildable.txt":"3","z.txt":"4"}
   """
  When I scan "." with detectors "notes:.txt"
  Then the scan finds "notes=a.txt,notes=z.txt"
  And the scan issues are "nested-skill=invalid.txt,invalid-skill=unbuildable.txt"

 @category/negative
 Scenario: A selected folder that isn't valid is not searched into either
  Given a source tree
   """
   {"invalid.dir/one.txt":"1","unbuildable.dir/two.txt":"2"}
   """
  When I scan "." with detectors "boxes:.dir,notes:.txt"
  Then the scan finds ""
  And the scan issues are "nested-skill=invalid.dir,invalid-skill=unbuildable.dir"

 @category/boundary
 Scenario: The .git folder is not searched
  Given a source tree
   """
   {".git/one.txt":"1","a/two.txt":"2"}
   """
  When I scan "." with detectors "notes:.txt"
  Then the scan finds "notes=a/two.txt"

 @category/error
 Scenario: A canceled scan reports it and finds nothing
  Given a source tree
   """
   {"a/one.txt":"1"}
   """
  And the scan is canceled
  When I scan "." with detectors "notes:.txt"
  Then the scan finds ""
  And the scan issues are "canceled=."

 @category/negative
 Scenario: Two detectors with one name are refused
  Given a source tree
   """
   {"a.txt":"1"}
   """
  When I scan "." with detectors "notes:.txt,notes:.md"
  Then the scanner is refused with "detector \"notes\" is registered twice"
