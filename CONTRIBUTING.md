### How to Contribute
  We enforce a strict **"Fork & Pull"** model for external contributors in order to reduce the friction to on board new contributors. New contributors can fork the repository, make their changes and submit a pull request which would be duly reviewed by the maintainers. If the pull request passes the review then it would be merged into the main source tree.

### Review Guidelines
  The following is the checklist that would be enforced on every single line of code that is to be merged into the main source tree:
* **Code should be readable**. Readable code is very important to us as we want the code to be as easily understood by any new developer that decides to contribute to our code. So it is mandatory that committers use meaningful variable names, method names and use [Go style coding conventions](https://github.com/golang/go/wiki/CodeReviewComments). Avoid never ending lines and provide proper indentation to long running lines of code. Run `make fmt` before submitting guide lines.
* **Appropriateness to the tool**. The new feature/ bug fix that is submitted for review must align itself to the vision of the tool.
* **Passing build**. It's of high importance that we always have a passing build to keep our source trees stable. Hence a passing build is mandatory. `make test` runs unit tests.
* **Add appropriate test cases**. Each fix/feature must have supporting test cases for the same to be merged back into the main source tree.
