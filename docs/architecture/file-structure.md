# File System Structure Documentation

This document outlines the organization of the Lokatik project's file system.

- ## [lokatik/](../..) (Project Root)
  - ## [.github/](../../.github)
    - _(Contains GitHub-specific configurations, workflows, issue templates, etc.)_
  - ## [assets/](../../assets)
    - ## [img/](../../assets/img)
      - ### [usecase.svg](../../assets/img/usecase.svg)
      - ### [wireflow.svg](../../assets/img/wireflow.svg)
  - ## [docs/](..)
    - ## [architecture/](../architecture)
      - **Purpose:** Contains documentation related to the system's high-level design, structure, and technical choices.
      - ## [adrs/](./adrs)
        - _(Architecture Decision Records)_
        - ### [template.md](./adrs/template.md) _(ADR template)_
      - ## [diagrams/](./diagrams)
        - ### [usecase.pdf](./diagrams/usecase.pdf)
        - ### [wireflow.pdf](./diagrams/wireflow.pdf)
      - ## [rfcs/](./rfcs)
        - _(Requests For Comments)_
        - ### [0001-Adopt-Flutter-for-Front-End.md](./rfcs/0001-Adopt-Flutter-for-Front-End.md)
        - ### [0002-Create-Boilerplate-for-Agile-Factory.md](./rfcs/0002-Create-Boilerplate-for-Agile-Factory.md)
        - ### [template.md](./rfcs/template.md) _(RFC template)_
      - ### [file-structure.md](file-structure.md) _(This file)_
      - ### [lokatik.drawio](lokatik.drawio)
      - ### [overview.md](overview.md)
    - ## [development/](../development)
      - **Purpose:** Guidelines and information for developers working on the project.
      - ### [conventions.md](../development/conventions.md) _(Coding style, naming, casing rules)_
  - ## [script/](../../scripts)
    - _(Utility scripts for automation, development, etc.)_
  - ### [.gitignore](../../.gitignore)
  - ### [LICENSE](../../LICENSE)
  - ### [README.md](../../README.md) _(Project root README)_
