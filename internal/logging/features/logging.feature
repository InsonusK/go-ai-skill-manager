@type/tech-check
Feature: Structured logging levels
 @category/happy
 Scenario Outline: Debug is opt-in
  When I log with debug "<debug>"
  Then log contains debug "<present>" and info "true"
  Examples:
   | debug | present |
   | false | false |
   | true | true |

 @category/happy
 Scenario Outline: Each standard log level has its own color
  When I log at level "<level>" with color "true"
  Then log level "<level>" has ANSI color "<code>"
  Examples:
   | level | code |
   | DEBUG | 90   |
   | INFO  | 36   |
   | WARN  | 33   |
   | ERROR | 31   |

 @category/happy
 Scenario: Color can be disabled
  When I log at level "WARN" with color "false"
  Then log contains ANSI "false"

 @category/happy
 Scenario Outline: Color mode accounts for the terminal and NO_COLOR
  When color mode is "<mode>", output terminal is "<terminal>" and NO_COLOR is "<no_color>"
  Then color is enabled "<enabled>"
  Examples:
   | mode   | terminal | no_color | enabled |
   | auto   | true     | false    | true    |
   | auto   | false    | false    | false   |
   | auto   | true     | true     | false   |
   | always | false    | false    | true    |
   | always | false    | true     | true    |
   | never  | true     | false    | false   |
