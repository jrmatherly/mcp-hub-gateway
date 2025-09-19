# AGENTS.md Refactoring Plan

## Current Issues

- File size: 930 lines (causing token limit warnings)
- Contains duplicated information from other documentation
- Includes outdated information from various phases
- Contains overly detailed implementation specifics

## Analysis of Current Content

### 1. Sections to REMOVE (Already documented elsewhere)

#### Build & Commands Section (Lines 29-109)

- **Reason**: Already documented in README.md and QUICKSTART.md
- **Action**: Remove entirely, reference those files instead

#### Detailed Code Style (Lines 110-196)

- **Reason**: Should be in CONTRIBUTING.md
- **Action**: Move to CONTRIBUTING.md if not already there

#### Testing Philosophy & Details (Lines 197-237)

- **Reason**: Already in testing-plan.md
- **Action**: Keep only essential testing reminders

#### Directory Structure Details (Lines 270-354)

- **Reason**: Already in README.md and implementation plan
- **Action**: Keep only critical directories for AI context

#### Configuration Details (Lines 355-420)

- **Reason**: Already in development-setup.md
- **Action**: Remove, reference development-setup.md

#### Docker Infrastructure Updates (Lines 456-532)

- **Reason**: Historical information, already in deployment guides
- **Action**: Remove entirely

#### Portal Implementation Status Details (Lines 533-681)

- **Reason**: Already in project-tracker.md and implementation plan
- **Action**: Keep only high-level summary

#### Phase Documentation Lists (Lines 708-755)

- **Reason**: Already in implementation plan README
- **Action**: Remove, reference implementation plan

#### Contributing Section (Lines 775-807)

- **Reason**: Already in CONTRIBUTING.md
- **Action**: Remove entirely

#### Quick Reference (Lines 808-839)

- **Reason**: Already in QUICKSTART.md
- **Action**: Remove entirely

#### Additional Resources (Lines 840-861)

- **Reason**: Already in README.md
- **Action**: Remove entirely

### 2. Sections to KEEP (Essential for AI assistants)

1. **Project Overview** (condensed)
2. **Critical Architecture Points**
3. **Security Considerations** (key points only)
4. **Current Development Priorities**
5. **Key Integration Points**
6. **Essential Testing Requirements**
7. **Important File Locations**

### 3. Sections to RELOCATE

1. **Code Style Conventions** → CONTRIBUTING.md
2. **Testing Philosophy** → docs/testing-philosophy.md
3. **Docker Setup Details** → docs/docker-setup.md
4. **Portal Development References** → implementation-plan/README.md

## Proposed New Structure for AGENTS.md

```markdown
# AGENTS.md

## Purpose

Brief guidance for AI coding assistants working in this repository.

## Project Overview

- Two projects: MCP Gateway CLI and MCP Portal
- Key architecture points
- Critical relationships

## Essential Context

- Portal wraps CLI (doesn't reimplement)
- Security-first approach
- Current phase and priorities

## Key Guidelines

- Command injection prevention
- Testing requirements (50% minimum)
- Parallel execution patterns
- Error handling patterns

## Important Locations

- Portal: /cmd/docker-mcp/portal/
- Implementation docs: /implementation-plan/
- Test files: alongside source

## Current Priorities

- Phase 4: Test coverage expansion
- Phase 5: OAuth planning

## References

- Details: See README.md, CONTRIBUTING.md
- Setup: See development-setup.md
- Implementation: See /implementation-plan/
```

## Target Size

- Current: 930 lines
- Target: < 200 lines
- Reduction: ~78%

## Implementation Steps

1. **Backup current AGENTS.md**
2. **Create/update CONTRIBUTING.md** with code style conventions
3. **Verify documentation completeness** in existing files
4. **Create streamlined AGENTS.md** with essential content only
5. **Update symlinks** if necessary (CLAUDE.md, etc.)
6. **Test with AI assistants** to ensure adequate context
