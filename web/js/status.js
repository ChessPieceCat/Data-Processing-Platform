document.addEventListener("DOMContentLoaded", () => {
    const activeStatuses = new Set(["queued", "processing"]);
    const jobCards = document.querySelectorAll(".job-card[data-job-id]");

    if (jobCards.length === 0) {
        return;
    }

    async function updateJobStatus(card) {
        const jobID = card.dataset.jobId;
        const statusElement = card.querySelector(".job-status");

        if (!jobID || !statusElement) {
            return false;
        }

        try {
            const response = await fetch(`/jobs/status?id=${encodeURIComponent(jobID)}`);

            if (!response.ok) {
                return false;
            }

            const data = await response.json();
            const newStatus = data.status;

            if (!newStatus) {
                return false;
            }

            const oldStatus = statusElement.textContent.trim();

            if (oldStatus !== newStatus) {
                statusElement.textContent = newStatus;

                statusElement.classList.remove(
                    "status-queued",
                    "status-processing",
                    "status-completed",
                    "status-failed"
                );

                statusElement.classList.add(`status-${newStatus}`);
            }

            return activeStatuses.has(newStatus);
        } catch (error) {
            console.error(`Error updating status for job ${jobID}:`, error);
            return false;
        }
    }

    async function updateStatuses() {
        const activeJobs = [];

        for (const card of jobCards) {
            const statusElement = card.querySelector(".job-status");

            if (!statusElement) {
                continue;
            }

            const currentStatus = statusElement.textContent.trim();

            if (activeStatuses.has(currentStatus)) {
                activeJobs.push(card);
            }
        }

        if (activeJobs.length === 0) {
            return;
        }

        const results = await Promise.all(
            activeJobs.map(updateJobStatus)
        );

        if (results.some(Boolean)) {
            setTimeout(updateStatuses, 3000);
        }
    }

    updateStatuses();
});