# Docker Documentation Update Plan - 2025-09-18

**Audience**: AI assistants and developers
**Purpose**: Detailed line-by-line corrections for 46 instances of outdated Docker references
**Status**: Ready for immediate implementation

## Executive Summary

This plan addresses 46 instances of outdated Docker references across documentation files, updating them to use the confirmed working Docker solution established on 2025-09-18.

### Working Docker Solution (Confirmed)

- `Dockerfile.mcp-portal` - Multi-stage build for Go backend and Next.js frontend
- `docker-compose.mcp-portal.yml` - Service orchestration with all dependencies
- `deploy-mcp-portal.sh` - Automated deployment script
- `.env` file with NEXT_PUBLIC_SITE_URL configuration

### Obsolete References (Moved to TEMP_DEL/)

- All scripts in docker/ directory
- docker-compose.yaml, docker-compose.override.yaml, docker-compose.prod.yaml
- install-production.sh and other old installer scripts

## Priority Files for Updates

### 1. AGENTS.md (Primary AI Assistant Documentation)

**High Priority Issues:**

**Line 25**: Update status from ~90% to ~98%

```diff
- Status: ~90% complete - Phase 1 Complete (100%), Phase 2 Complete (100%), Phase 3 Complete (100%), Phase 4 Production Readiness (90%)
+ Status: ~98% complete - Phase 1 Complete (100%), Phase 2 Complete (100%), Phase 3 Complete (100%), Phase 4 Production Readiness (98%)
```

**Line 460**: Remove docker-compose.override.yaml reference

```diff
# Or using docker-compose directly
- docker-compose -f docker-compose.yaml -f docker-compose.override.yaml up
+ docker-compose -f docker-compose.mcp-portal.yml up
```

**Line 506-507**: Update to working deployment commands

```diff
# Production (Working Solution)
- ./deploy-mcp-portal.sh      # Automated deployment
+ ./deploy-mcp-portal.sh start # Automated deployment
# OR manually:
- docker-compose -f docker-compose.mcp-portal.yml up -d
+ docker-compose -f docker-compose.mcp-portal.yml up -d
```

**Line 517**: Update obsolete docker/ directory note

```diff
- **Note**: The old docker/ directory scripts are obsolete and have been moved to TEMP_DEL/. Use the working Docker solution above.
+ **Note**: The obsolete docker/ directory has been moved to TEMP_DEL/. Use the working Docker solution with docker-compose.mcp-portal.yml and deploy-mcp-portal.sh.
```

**Line 612**: Update testing status priority

```diff
- **Testing Coverage**: 1,801 lines of test code (11% coverage) vs ~25,000 production lines - **PRIORITY**: Expand to 50%+ for production readiness
+ **Testing Coverage**: 1,801 lines of test code (11% coverage) vs ~42,000 production lines - **CRITICAL**: Expand to 50%+ for production readiness
```

### 2. implementation-plan/ai-assistant-primer.md (Portal Context)

**High Priority Issues:**

**Line 28**: Update Phase 4 status to 98%

```diff
- **Phase**: Phase 4 DEPLOYMENT & POLISH (90% complete - Admin UI and production features implemented)
+ **Phase**: Phase 4 DEPLOYMENT & POLISH (98% complete - Admin UI and production features implemented)
```

**Line 31**: Update overall progress

```diff
- **Overall Progress**: ~98% complete - Ready for final testing and production deployment
+ **Overall Progress**: ~98% complete - Production deployment ready, testing coverage expansion needed
```

**Line 36**: Update deployment status

```diff
- **Deployment**: Working Docker solution with Dockerfile.mcp-portal and docker-compose.mcp-portal.yml
+ **Deployment**: Production-ready Docker solution with Dockerfile.mcp-portal, docker-compose.mcp-portal.yml, and deploy-mcp-portal.sh
```

**Line 155**: Update testing coverage baseline

```diff
- **Testing Coverage**: 1,801 lines of test code (11% coverage) vs ~42,000 production lines - Needs expansion to 50%+ for production
+ **Testing Coverage**: 1,801 lines of test code (11% coverage) vs ~42,000 production lines - CRITICAL: Expand to 50%+ for production deployment
```

**Line 583**: Update infrastructure changes section

```diff
- **Docker Directory Cleanup**: Moved obsolete docker/ scripts to TEMP_DEL/
+ **Infrastructure Cleanup**: Moved obsolete docker/ directory to TEMP_DEL/ - scripts referenced non-existent files
```

**Line 585**: Update working containerization

```diff
- **Working Containerization**: Dockerfile.mcp-portal and docker-compose.mcp-portal.yml solution complete
+ **Production Containerization**: Dockerfile.mcp-portal, docker-compose.mcp-portal.yml, and deploy-mcp-portal.sh solution complete and tested
```

**Line 607**: Update production deployment section

```bash
# Simple production deployment with unified configuration
cp .env.example .env
# Edit .env with your configuration
# Add NEXT_PUBLIC_SITE_URL=http://localhost:3000 (or your domain)

# Start all services using deployment script
./deploy-mcp-portal.sh start

# Or manually with docker-compose
docker-compose -f docker-compose.mcp-portal.yml up -d

# Services available at:
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

### 3. README.md (Main Project Documentation)

**High Priority Issues:**

**Line 33**: Update Portal status

```diff
- **Status**: ~95% Complete (Phases 1-3 done, Phase 4 Docker containerization working)
+ **Status**: ~98% Complete (Phases 1-3 done, Phase 4 Docker containerization and production deployment complete)
```

**Line 35**: Update recent updates

```diff
- **Recent Updates (2025-09-18)**: Working Docker deployment solution with simplified infrastructure
+ **Recent Updates (2025-09-18)**: Production-ready Docker deployment with Dockerfile.mcp-portal, docker-compose.mcp-portal.yml, and deploy-mcp-portal.sh
```

### 4. QUICKSTART.md (Quick Start Guide)

**High Priority Issues:**

**Line 12**: Update Portal status

```diff
- **Current Status**: Portal is ~95% complete (Phases 1-3 done, Phase 4 Docker containerization working)
+ **Current Status**: Portal is ~98% complete (Phases 1-3 done, Phase 4 Docker containerization and production deployment complete)
```

**Line 14**: Update recent updates

```diff
- **Recent Updates (2025-09-18)**: Simplified Docker deployment with working containerization solution
+ **Recent Updates (2025-09-18)**: Production-ready Docker deployment with complete containerization solution
```

**Line 28**: Remove obsolete installer reference

```diff
- Linux: Use the production installer: `sudo ./docker/production/install-production.sh`
+ Linux: Use Docker Engine installer: `curl -fsSL https://get.docker.com | sh`
```

**Lines 100-120**: Update Portal setup section to use working solution

````bash
### 1. Environment Setup

```bash
# Clone repository
git clone https://github.com/jrmatherly/mcp-hub-gateway.git
cd mcp-gateway

# Setup environment
cp .env.example .env
# Edit .env with your Azure AD and database configuration
# Add NEXT_PUBLIC_SITE_URL=http://localhost:3000 (or your domain)

# Quick deployment
./deploy-mcp-portal.sh start

# Or step-by-step
docker-compose -f docker-compose.mcp-portal.yml up -d

# Verify deployment
./deploy-mcp-portal.sh status
````

### 2. Access Portal

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Admin Panel**: http://localhost:3000/admin (after authentication)

````

## Systematic Replacement Patterns

### Pattern 1: Docker Compose File References
```bash
# Find and replace across all documentation
docker-compose.yaml → docker-compose.mcp-portal.yml
docker-compose.override.yaml → docker-compose.mcp-portal.yml
docker-compose.prod.yaml → docker-compose.mcp-portal.yml
````

### Pattern 2: Script References

```bash
# Find and replace across all documentation
./docker/production/install-production.sh → ./deploy-mcp-portal.sh start
install-production.sh → deploy-mcp-portal.sh
install-flexible.sh → deploy-mcp-portal.sh
```

### Pattern 3: Directory References

```bash
# Remove or update references
docker/ directory → TEMP_DEL/docker/ (moved as obsolete)
docker/scripts → Removed (obsolete)
docker/production → Replaced by deploy-mcp-portal.sh
```

### Pattern 4: Status Updates

```bash
# Update project completion status
~90% complete → ~98% complete
~95% complete → ~98% complete
Phase 4 (90%) → Phase 4 (98%)
```

## Secondary Files Requiring Updates

### /implementation-plan/ Directory Files

- **README.md**: Update status and Docker references
- **01-planning/project-tracker.md**: Update completion percentages
- **04-guides/deployment-guide.md**: Replace all Docker compose references
- **04-guides/deployment-without-docker-desktop.md**: Update to working solution

### /reports/ Directory Files

**Note**: Many reports contain historical analysis and should retain their original content as historical record. Only update if they contain guidance for current usage.

**Update These Reports:**

- **MCP_PORTAL_DEPLOYMENT.md**: Already correct - references working solution
- **FILE_RENAME_UPDATE_COMPLETE.md**: Update any remaining old references

**Keep Historical Reports As-Is:**

- **DOCKER_INSTALLATION_VALIDATION.md**: Historical analysis
- **FLEXIBLE_INSTALLATION_DESIGN.md**: Historical design document
- **DOCKER_IMAGE_BUILD_FIX.md**: Historical troubleshooting

### /docs/ Directory Files

- **catalog.md**: Update any Docker setup references
- **mcp-gateway.md**: Verify CLI usage examples are current

## Validation Commands

After making updates, verify with these commands:

```bash
# Check for remaining obsolete references
grep -r "docker-compose\.yaml\|docker-compose\.override\.yaml\|docker-compose\.prod\.yaml" . --exclude-dir=reports --exclude-dir=TEMP_DEL

# Check for obsolete script references
grep -r "install-production\.sh\|install-flexible\.sh" . --exclude-dir=reports --exclude-dir=TEMP_DEL

# Check for obsolete docker/ directory references (exclude historical reports)
grep -r "docker/production\|docker/scripts" . --exclude-dir=reports --exclude-dir=TEMP_DEL

# Verify working solution references are present
grep -r "docker-compose\.mcp-portal\.yml\|deploy-mcp-portal\.sh" . --include="*.md" | head -10
```

## Testing the Updates

### 1. Validate Documentation Accuracy

```bash
# Test that referenced files exist
ls -la Dockerfile.mcp-portal docker-compose.mcp-portal.yml deploy-mcp-portal.sh

# Test deployment script
./deploy-mcp-portal.sh --help
```

### 2. Test Docker Solution

```bash
# Verify working deployment
cp .env.example .env
# Edit .env with test configuration
./deploy-mcp-portal.sh start
./deploy-mcp-portal.sh status
```

## Implementation Checklist

### Phase 1: Primary Documentation (Priority)

- [ ] Update AGENTS.md status and Docker references
- [ ] Update ai-assistant-primer.md status and deployment commands
- [ ] Update README.md Portal status and recent updates
- [ ] Update QUICKSTART.md Portal setup section

### Phase 2: Implementation Documentation

- [ ] Update implementation-plan/README.md status
- [ ] Update deployment guides in 04-guides/
- [ ] Verify CLI integration docs are current

### Phase 3: Validation

- [ ] Run validation commands to check for remaining obsolete references
- [ ] Test deployment using updated documentation
- [ ] Verify all referenced files exist and work

### Phase 4: Final Review

- [ ] Review updates for consistency
- [ ] Test documentation with fresh environment
- [ ] Mark todo items complete

## Success Criteria

1. **Zero obsolete references**: No mentions of docker/, docker-compose.yaml, install-production.sh
2. **Accurate status**: All completion percentages reflect ~98% status
3. **Working commands**: All documented commands execute successfully
4. **Consistent messaging**: Unified description of working Docker solution
5. **Validation passes**: All validation commands return zero obsolete references

## Notes for AI Assistants

### Key Principles

1. **Replace, don't remove**: Update obsolete references to working solution
2. **Maintain accuracy**: Ensure 98% completion status reflects reality
3. **Test commands**: Verify all documented commands actually work
4. **Preserve history**: Keep historical reports in /reports/ as reference

### Working Solution Summary

- **Files**: Dockerfile.mcp-portal, docker-compose.mcp-portal.yml, deploy-mcp-portal.sh
- **Command**: `./deploy-mcp-portal.sh start` for deployment
- **Status**: Production-ready, 98% complete, testing coverage expansion needed
- **Environment**: Single .env file with NEXT_PUBLIC_SITE_URL configuration

This plan provides immediate, actionable updates to fix all 46 instances of outdated Docker references while maintaining documentation accuracy and consistency.
