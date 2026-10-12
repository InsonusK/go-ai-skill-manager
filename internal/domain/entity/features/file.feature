@type/domain
Feature: Access a skill file through the File contract
 @category/happy
 Scenario: Content reads a file once and caches its bytes
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I read the contract file content
  Then the contract file content is "intro"
  And the contract file was read 1 time
  When I read the contract file content
  Then the contract file content is "intro"
  And the contract file was read 1 time

 @category/happy
 Scenario: Path returns a path relative to the owning skill
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "skill-relative"
  Then the requested file path is "docs/intro.md"

 @category/happy
 Scenario: Path returns a path relative to the repository
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "repo-absolute"
  Then the requested file path is "a/docs/intro.md"

 @category/happy
 Scenario: Path returns an absolute filesystem path
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "os-absolute"
  Then the requested file path is "/source/a/docs/intro.md"

 @category/negative
 Scenario: Path refuses a path relative to the file itself
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "file-relative"
  Then requesting the file path fails with "unsupported-path-kind"

 @category/negative
 Scenario: Path rejects an unknown path kind
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  When I request the contract file path as "unknown"
  Then requesting the file path fails with "unknown path kind"

 @category/happy
 Scenario: Skill returns the file's owning skill
  Given a file "docs/intro.md" containing "intro" attached to a mocked skill rooted at "a" in repository "/source"
  Then the contract file belongs to the mocked skill

 @category/happy
 Scenario: Links returns the file's discovered links
  Given a file "docs/intro.md" containing "[Guide](./guide.md#intro)" attached to a mocked skill rooted at "a" in repository "/source"
  When I read the contract file links
  Then the contract file link targets are "./guide.md"

 @category/happy
 Scenario: Links reads the file's content once and caches the discovered links
  Given a file "docs/intro.md" containing "[Guide](./guide.md#intro)" attached to a mocked skill rooted at "a" in repository "/source"
  When I read the contract file links
  And I read the contract file links
  Then the contract file was read 1 time

 @category/negative
 Scenario: Links fails clearly when no LinkSearcher is configured
  Given a file "docs/intro.md" containing "[Guide](./guide.md#intro)" attached to a mocked skill rooted at "a" in repository "/source"
  And no link searcher is configured
  When I read the contract file links
  Then reading the contract file links fails with "no LinkSearcher configured"
