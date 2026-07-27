# Claude instructions

## Documentation

Read `docs/STYLE.md` before you create or edit documentation.

Apply these rules:

1. Put project documentation under `docs/`.
2. Add each document to `docs/README.md`.
3. State the purpose or outcome in the first paragraph.
4. Use active voice.
5. Give each sentence and paragraph one topic.
6. Limit procedure steps to 20 words.
7. Limit descriptive sentences to 25 words.
8. Give one instruction in each procedure step.
9. Start each procedure step with an imperative verb.
10. Use one term for each concept.
11. Define abbreviations when they first occur.
12. Remove words that do not add information.
13. Do not use idioms, stale metaphors, or promotional language.
14. Keep commands, identifiers, API fields, and error text exact.
15. Include verification and recovery sections in procedures.

Use these rules as a practical subset of ASD-STE100 and Orwell's writing
rules. Do not claim full ASD-STE100 compliance.

## Commits

Write the subject line and nothing else. Run `git log --oneline -20` for
examples.

Apply these rules:

1. Write the subject as `type(scope): summary`.
2. Use one of `build`, `chore`, `ci`, `docs`, `feat`, `fix`, or `refactor`.
3. Name the scope after the package or area, in lowercase. Recent examples are
   `api`, `serve`, `dev`, `ingress`, `images`, `web`, `cron`, `cache`, `deps`,
   and `logging`.
4. Omit the scope when the change covers the whole repository.
5. Start the summary with a lowercase imperative verb.
6. Limit the subject to 65 characters.
7. Do not put a period at the end of the subject.
8. Do not write a body. The subject is the whole message.
9. Do not add trailers. This includes `Co-Authored-By`, generated-with
   notices, and session links.
10. Put one concern in each commit. Split unrelated work into separate commits.
11. Let the `lefthook` hooks run. Do not pass `--no-verify`.

Commit to `main`. The history is linear and holds no merge commits.

The documentation rules above also apply to a subject line. Prefer the short
word, and remove each word that does not add information.
