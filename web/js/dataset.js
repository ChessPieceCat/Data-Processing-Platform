const csvFileInput = document.getElementById("csvFile");
const inspectButton = document.getElementById("inspectDataset");

const datasetInfo = document.getElementById("datasetInfo");
const datasetStatus = document.getElementById("datasetStatus");

const submitJobButton = document.getElementById("submitJob");

const modelSelect = document.getElementById("model");
const modelOptions = document.getElementById("modelOptions");

const modelVisualizationOptions = document.getElementById("modelVisualizationOptions");

const configTypeSelect = document.getElementById("configType");
const manualConfig = document.getElementById("manualConfig");

const targetSelect = document.getElementById("target");
const featureSelectionInputs = document.querySelectorAll(
    'input[name="featureSelection"]'
);

const manualFeatures = document.getElementById("manualFeatures");
const featureList = document.getElementById("featureList");


function updateModelOptions() {
    const modelSelected = modelSelect.value !== "none";

    modelOptions.hidden = !modelSelected;
    modelVisualizationOptions.hidden = !modelSelected;

    if (!modelSelected) {
        manualConfig.hidden = true;
        return;
    }

    updateConfigOptions();
}


function updateConfigOptions() {
    const manual = configTypeSelect.value === "manual";

    manualConfig.hidden = !manual;
}


function updateFeatureOptions() {
    const selected = document.querySelector(
        'input[name="featureSelection"]:checked'
    );

    const manual = selected && selected.value === "manual";

    manualFeatures.hidden = !manual;
}


function populateColumns(columns) {
    // Clear existing target options.
    targetSelect.innerHTML = "";

    const defaultOption = document.createElement("option");
    defaultOption.value = "";
    defaultOption.textContent = "Select a target";

    targetSelect.appendChild(defaultOption);

    // Clear existing feature options.
    featureList.innerHTML = "";

    columns.forEach((column) => {
        // Add target option.
        const targetOption = document.createElement("option");
        targetOption.value = column;
        targetOption.textContent = column;

        targetSelect.appendChild(targetOption);

        // Add feature checkbox.
        const label = document.createElement("label");

        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.name = "features";
        checkbox.value = column;

        label.appendChild(checkbox);
        label.appendChild(document.createTextNode(` ${column}`));

        featureList.appendChild(label);
        featureList.appendChild(document.createElement("br"));

        checkbox.addEventListener("change", updateFeatureAvailability);
    });

    targetSelect.addEventListener("change", updateFeatureAvailability);

    updateFeatureAvailability();
}


function updateFeatureAvailability() {
    const target = targetSelect.value;

    const featureCheckboxes = document.querySelectorAll(
        'input[name="features"]'
    );

    featureCheckboxes.forEach((checkbox) => {
        const isTarget = checkbox.value === target;

        if (isTarget) {
            checkbox.checked = false;
            checkbox.disabled = true;
        } else {
            checkbox.disabled = false;
        }
    });
}


inspectButton.addEventListener("click", async () => {
    const file = csvFileInput.files[0];

    if (!file) {
        datasetStatus.textContent = "Please select a CSV file first.";
        datasetInfo.hidden = false;
        return;
    }

    const formData = new FormData();
    formData.append("csvFile", file);

    datasetStatus.textContent = "Inspecting dataset...";
    datasetInfo.hidden = false;
    submitJobButton.disabled = true;

    try {
        const response = await fetch("/inspect/dataset", {
            method: "POST",
            body: formData
        });

        if (!response.ok) {
            const message = await response.text();
            throw new Error(message);
        }

        const data = await response.json();

        document.getElementById("uploadID").value = data.upload_id;

        populateColumns(data.columns);

        datasetStatus.textContent =
            `Dataset loaded successfully. ${data.columns.length} columns found.`;

        submitJobButton.disabled = false;

    } catch (error) {
        datasetStatus.textContent =
            `Error inspecting dataset: ${error.message}`;

        submitJobButton.disabled = true;
    }
});


modelSelect.addEventListener("change", updateModelOptions);

configTypeSelect.addEventListener("change", updateConfigOptions);

featureSelectionInputs.forEach((input) => {
    input.addEventListener("change", updateFeatureOptions);
});


// Establish the correct initial state when the page loads.
updateModelOptions();
updateFeatureOptions();