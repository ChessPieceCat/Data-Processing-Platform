const csvFileInput = document.getElementById("csvFile");
const inspectButton = document.getElementById("inspectDataset");
const datasetInfo = document.getElementById("datasetInfo");
const datasetStatus = document.getElementById("datasetStatus");
const submitJobButton = document.getElementById("submitJob");
const modelSelect = document.getElementById("model");
const modelOptions = document.getElementById("modelOptions");
const modelVisualizationOptions = document.getElementById(
    "modelVisualizationOptions"
);
const configTypeSelect = document.getElementById("configType");
const manualConfig = document.getElementById("manualConfig");
const targetSelect = document.getElementById("target");
const featureSelectionInputs = document.querySelectorAll(
    'input[name="featureSelection"]'
);
const manualFeatures = document.getElementById("manualFeatures");
const featureList = document.getElementById("featureList");

// Image processing elements.
const imageFileInput = document.getElementById("imageFile");
const imageOptions = document.getElementById("imageOptions");
const imageUploadID = document.getElementById("imageUploadID");
const resizeImageCheckbox = document.getElementById("resizeImage");
const resizeOptions = document.getElementById("resizeOptions");
const compressImageCheckbox = document.getElementById("compressImage");
const compressionOptions = document.getElementById("compressionOptions");
const convertFormatCheckbox = document.getElementById("convertFormat");
const formatOptions = document.getElementById("formatOptions");
const outputFormat = document.getElementById("outputFormat");


// Upload the selected image and show processing options.
if (imageFileInput && imageOptions) {
    imageFileInput.addEventListener("change", async () => {
        const file = imageFileInput.files[0];

        if (!file) {
            imageOptions.hidden = true;
            imageUploadID.value = "";
            return;
        }

        imageOptions.hidden = true;

        try {
            const formData = new FormData();
            formData.append("imageFile", file);

            const response = await fetch("/upload/image", {
                method: "POST",
                body: formData
            });

            if (!response.ok) {
                const message = await response.text();
                throw new Error(message);
            }

            const data = await response.json();

            imageUploadID.value = data.upload_id;
            imageOptions.hidden = false;
        } catch (error) {
            imageOptions.hidden = true;
            imageFileInput.value = "";

            alert(`Error uploading image: ${error.message}`);
        }
    });
}


// Show or hide resize options.
if (resizeImageCheckbox && resizeOptions) {
    resizeImageCheckbox.addEventListener("change", () => {
        resizeOptions.hidden = !resizeImageCheckbox.checked;
    });
}


// Show or hide compression options.
if (compressImageCheckbox && compressionOptions) {
    compressImageCheckbox.addEventListener("change", () => {
        updateImageCompressionOptions();
    });
}

// Update the visibility of compression options based on the selected output format.
if (outputFormat) {
    outputFormat.addEventListener(
        "change",
        updateImageCompressionOptions
    );
}


// Show or hide format-conversion options.
if (convertFormatCheckbox && formatOptions) {
    convertFormatCheckbox.addEventListener("change", () => {
        formatOptions.hidden = !convertFormatCheckbox.checked;
        updateImageCompressionOptions();
    });
}


// Establish the correct initial state when the page loads.
updateModelOptions();
updateFeatureOptions();

if (imageOptions) {
    imageOptions.hidden =
        !imageFileInput || imageFileInput.files.length === 0;
}

if (resizeOptions) {
    resizeOptions.hidden =
        !resizeImageCheckbox || !resizeImageCheckbox.checked;
}

if (formatOptions) {
    formatOptions.hidden =
        !convertFormatCheckbox || !convertFormatCheckbox.checked;
}

updateImageCompressionOptions();

// Update the visibility of compression options based on the selected output format.
function updateImageCompressionOptions() {
    if (!compressionOptions || !compressImageCheckbox) {
        return;
    }

    if (!compressImageCheckbox.checked) {
        compressionOptions.hidden = true;
        return;
    }

    const file = imageFileInput?.files[0];

    const originalIsPng =
        file && file.type === "image/png";

    const convertingToPng =
        convertFormatCheckbox &&
        convertFormatCheckbox.checked &&
        outputFormat &&
        outputFormat.value === "png";

    if (originalIsPng && !convertFormatCheckbox.checked) {
        compressionOptions.hidden = true;
        return;
    }

    if (convertingToPng) {
        compressionOptions.hidden = true;
        return;
    }

    compressionOptions.hidden = false;
}