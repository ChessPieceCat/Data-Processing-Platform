# Project Problems and Solutions

This document records significant problems encountered while developing and deploying the Data Processing Platform, the investigation used to identify their causes, the solutions implemented, and the lessons learned.

## 1. Deployment used a stale application image tag

### Problem
The AWS deployment initially used a stale or hard-coded Docker image tag, so deployments did not reliably run the image associated with the triggering commit.

### Investigation
The deployment behavior was inspected and the relationship between the EC2 checkout, the Git commit, and the ECR image tag was found to be inconsistent.

### Solution
The SSM deployment was changed to synchronize the EC2 repository with `origin/main` before deploying and to pull the application image associated with the triggering commit.

### Lesson
A deployment pipeline should make the relationship between source revision, image tag, and deployed revision explicit.

---

## 2. ARM64 and AMD64 Docker image mismatch

### Problem
The EC2 deployment target is ARM64, while an application image was initially built for AMD64. Docker warned about the platform mismatch and the application and worker containers failed with `exec format error`.

### Investigation
Compose output showed the requested image was `linux/amd64` while the host was `linux/arm64/v8`. Container logs then showed `exec ./server: exec format error` and the equivalent worker error.

### Solution
The CI/CD pipeline was changed to build and publish ARM64 application images to match the ARM64 EC2 instance.

### Lesson
The deployment target architecture must be treated as part of the build configuration rather than assumed.

---

## 3. Session Manager initially would not provide an interactive shell

### Problem
The EC2 Session Manager browser connection created a session but remained on a blank screen. The AWS CLI also reported:

```text
Plugin with name Standard_Stream not found
```

### Investigation
The session was confirmed to be created successfully. The local Session Manager plugin was checked and was initially missing.

### Solution
The Session Manager plugin was installed locally so that `aws ssm start-session` could provide an interactive shell. The remaining connection problem was later traced to the EC2 instance's full filesystem; once disk space was recovered, the SSM session worked normally.

### Lesson
When troubleshooting connectivity, distinguish client-side tooling failures from failures occurring on the managed instance.

---

## 4. EC2 root filesystem reached 100% usage

### Problem
The production application began returning `500 Internal Server Error` responses when users attempted to upload files.

### Investigation
Because dataset, image, and route uploads were all affected, a shared infrastructure problem was suspected rather than a processor-specific bug. CloudWatch showed disk usage at 100%. `df -h` confirmed the root filesystem had no free space.

The application log provided the direct cause:

```text
Error creating temporary upload directory: mkdir uploads: no space left on device
```

Filesystem usage was then traced from `/var` to `/var/lib` to `/var/lib/containerd`.

### Solution
Unused Docker images were identified and removed with:

```bash
docker image prune -a -f
```

This reclaimed 1.21 GB and reduced root filesystem usage from 100% to 67%. Uploads immediately began working again.

### Lesson
When several application features fail simultaneously, inspect shared system resources such as disk, memory, networking, and process state before debugging each application path independently.

---

## 5. Repeated deployments accumulated old Docker images

### Problem
The disk outage was caused by repeated deployments leaving old application images on the EC2 host. The production instance had a 15 GB root volume, while the application image was roughly 1 GB.

### Investigation
`docker system df -v` showed multiple older application images with approximately 800 MB of unique storage each and zero containers using them. Only the current image had active application and worker containers.

### Solution
Add the following command to the deployment process after the new containers have been recreated successfully:

```bash
docker image prune -a -f
```

This removes unused images while preserving images still referenced by running containers.

### Lesson
Deployment automation should include lifecycle cleanup, not just image retrieval and container replacement. Operational resources need to be managed as part of the deployment process.

---

## 6. SSM and application failures had the same underlying cause

### Problem
Several infrastructure and application symptoms appeared at the same time:

- application uploads returned HTTP 500
- Session Manager would not provide an interactive shell
- Run Command failed

### Investigation
The failures were initially treated as separate problems. Once filesystem usage was inspected, the root filesystem was found to be completely full. The application's log explicitly reported `no space left on device`.

### Solution
Removing unused Docker images restored filesystem capacity. Afterward, application uploads worked and SSM sessions became usable again.

### Lesson
When multiple independent services fail together, look for a shared infrastructure bottleneck before treating each failure as unrelated.

---

## 7. Temporary uploads required ownership protection

### Problem
After introducing guest sessions and registered users, temporary uploads also needed to respect the current identity. Otherwise, ownership checks applied only to permanent jobs could leave a gap in the upload workflow.

### Investigation
The full lifecycle of an uploaded file was reviewed rather than checking only the final job record.

### Solution
Temporary uploads were associated with the current session and validated against that session before submission.

### Lesson
Authorization should cover an object's entire lifecycle, including temporary state and intermediate resources.

---

## 8. Guest sessions needed to coexist with registered accounts

### Problem
The platform needed to support anonymous use while still providing private job histories for registered users.

### Investigation
A server-side guest-session model was introduced. This created a follow-on question: what should happen to jobs created before a guest registers for an account?

### Solution
Guest jobs are associated with the guest session. When the guest registers, those jobs are transferred to the newly created account.

### Lesson
A shared ownership model based on identity makes guest and authenticated workflows easier to support without duplicating the job system.

---

## 9. Cross-user and cross-session job access had to be prevented

### Problem
A job ID alone must not be sufficient to access another user's results or artifacts.

### Investigation
Job-related endpoints were reviewed for cases where a caller could supply a different job ID and bypass ownership restrictions.

### Solution
Request handlers use an ownership-aware retrieval path before returning job results or downloadable artifacts. Authorization is performed against the current user or guest session.

### Lesson
Authentication establishes identity; authorization determines whether that identity may access a particular resource.

---

## 10. Homepage job status required live updates

### Problem
The homepage displayed job status, but users had to refresh the entire page to see transitions from `queued` to `processing` to `completed` or `failed`.

### Investigation
The application already exposed HTTP endpoints and did not require a full real-time messaging system for the small amount of status information involved.

### Solution
A small status endpoint and vanilla JavaScript `fetch()` polling were added. The browser polls only queued or processing jobs and updates only the corresponding status badge. Polling stops when jobs reach terminal states.

### Lesson
Use the simplest mechanism that satisfies the actual requirement. HTTP polling was sufficient without adding WebSockets or another persistent real-time channel.

---

## 11. Result pages initially had inconsistent visual structure

### Problem
Dataset, image, and route result pages used different markup patterns and therefore had inconsistent visual presentation.

### Investigation
The result pages were compared as a group rather than styling each page independently.

### Solution
Shared semantic classes were introduced, including:

- `result-section`
- `result-grid`
- `result-item`
- `result-list`
- `result-table`
- `result-images`
- `result-actions`

These classes allow shared CSS components to provide a consistent design across job types.

### Lesson
Shared component structure makes frontend styling and maintenance more predictable than page-specific element styling.

---

## 12. Dataset visualizations consumed too much vertical space

### Problem
Multiple large visualization images stacked vertically and dominated the dataset results page.

### Investigation
The visualizations were independent artifacts that did not need to occupy the full width of the page one after another.

### Solution
The visualization area was changed to a responsive grid with multiple columns on wider screens and a single column on small screens.

### Lesson
Layout should reflect the relationship between pieces of content rather than simply following the order in which they appear in the source document.

---

## 13. Public demo needed sample inputs

### Problem
A public application that requires visitors to prepare their own CSVs or images creates unnecessary friction during evaluation.

### Investigation
The application is intended to be publicly demonstrated, so a visitor should be able to try the processors immediately.

### Solution
Downloadable sample dataset, image, and route files were added to the homepage. They are served through the existing static file server rather than through additional backend endpoints.

### Lesson
A public demo should minimize the distance between discovering the project and successfully using it.

---

## 14. Frontend changes were baked into the Docker image

### Problem
Local HTML and CSS changes did not automatically appear in the running application container because the web assets were copied into the Docker image instead of being bind-mounted.

### Investigation
The container configuration was checked to determine whether the web directory was mounted from the host.

### Solution
Local frontend changes require rebuilding the application image when using the normal Compose configuration. Production deployment uses versioned images through CI/CD.

### Lesson
Understand which application state is bind-mounted at runtime and which is baked into the image when developing and debugging containerized applications.

---

## 15. Public network boundaries needed tightening

### Problem
The initial deployment exposed the application directly rather than making a reverse proxy the sole public entry point.

### Investigation
The EC2 security group and Compose port mappings were reviewed as part of the production security work.

### Solution
Caddy was introduced as the public reverse proxy on ports 80 and 443. The Go application port and monitoring/queue services remain private.

### Lesson
A service being reachable from the Internet is not the same thing as being appropriately exposed to the Internet. Public and private boundaries should be deliberately designed and verified.

---

## Debugging Pattern

Several of the most useful debugging sessions followed the same general process:

```text
symptom
  ↓
narrow the scope
  ↓
inspect the next layer down
  ↓
collect objective evidence
  ↓
identify the underlying cause
  ↓
fix the cause
  ↓
verify the result
  ↓
automate prevention where appropriate
```

The disk-exhaustion incident is the clearest example:

```text
uploads fail
  ↓
all upload types fail
  ↓
suspect shared infrastructure
  ↓
CloudWatch shows 100% disk usage
  ↓
`df -h` confirms no free space
  ↓
`/var` is unusually large
  ↓
`/var/lib/containerd` is the major consumer
  ↓
Docker shows old application images
  ↓
remove unused images
  ↓
disk usage falls from 100% to 67%
  ↓
uploads work again
  ↓
SSM works again
  ↓
automate image cleanup in deployment
```

The goal is not only to restore service when a failure occurs, but to use the root-cause analysis to improve the system so the same failure mode is less likely to recur.
