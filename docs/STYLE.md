# Documentation style

This guide defines how contributors write Shrike documentation.

## Standard

Use Simplified Technical English principles for technical content.

Use Orwell's rules to remove vague, inflated, or unnecessary language.

This project does not claim full ASD-STE100 compliance. Full compliance requires
the complete controlled dictionary and all writing rules.

The official references are:

- [ASD-STE100 Issue 9](https://www.asd-ste100.org/assets/files/ASD-STE100_ISSUE9.pdf)
- [Politics and the English Language](https://www.orwellfoundation.com/the-orwell-foundation/orwell/essays-and-other-%20works/politics-and-the-english-language/)

## Required rules

1. State the purpose or outcome in the first paragraph.
2. Use active voice unless the actor is unknown.
3. Give each sentence one topic.
4. Give each paragraph one topic.
5. Use no more than six sentences in one paragraph.
6. Use no more than 25 words in a descriptive sentence.
7. Use no more than 20 words in a procedure step.
8. Give one instruction in each procedure step.
9. Start each procedure step with an imperative verb.
10. Put a required condition before its action.
11. Use one term for each concept.
12. Define an abbreviation when it first occurs.
13. Prefer a common short word over a long word.
14. Remove each word that does not add information.
15. Do not use idioms, stale metaphors, humor, or promotional language.
16. Do not use `simply`, `just`, `obviously`, or `easy`.
17. Use American English spelling.
18. Keep code names, API fields, commands, and error text exact.
19. Break a style rule when the rule makes technical information less accurate.

Headings, code, tables, links, and identifiers do not count toward sentence
length.

## Document types

Choose one document type before you write.

### Procedure

Use a procedure when the reader must complete a task.

Use this structure:

```text
# Task name

State the result.

## Prerequisites
## Procedure
## Verification
## Recovery
```

### Reference

Use a reference when the reader needs exact facts or contracts.

Use this structure:

```text
# Subject

State the scope.

## Contract
## Defaults
## Failure modes
## Source
```

### Decision

Use a decision document when a choice affects future implementation.

Use this structure:

```text
# Decision

Status: Draft, Accepted, or Superseded

## Context
## Decision
## Consequences
```

## Review checklist

- The first paragraph states the result or purpose.
- Commands and configuration names match the current implementation.
- Each procedure includes a verification step.
- Each destructive procedure includes a recovery step.
- Links point to the source of truth.
- The document follows the sentence limits.
- The change updates the index when it adds or removes a document.
