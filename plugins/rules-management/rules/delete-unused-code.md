After any refactor or change that removes a production caller, check whether the
affected function, method, type, constant, or import is still referenced anywhere
in non-test code. If it is only referenced by its own tests (or not at all),
delete both the code and the tests. Do not leave dead production code behind just
because tests exist for it — tests that only exercise unreachable code provide
false confidence.

Steps:
1. Search the entire codebase for every call site of the removed symbol.
2. If all remaining references are in test files, delete the symbol and its tests.
3. Remove any imports that become unused as a result.
4. Build and run tests to confirm nothing breaks.
