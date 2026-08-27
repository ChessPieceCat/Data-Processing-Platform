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

// Route processing elements.
const routeFileInput = document.getElementById("routeFile");
const distanceFileInput = document.getElementById("distanceFile");
const routeUploadID = document.getElementById("routeUploadID");
const routeOptions = document.getElementById("routeOptions");
const startLocationSelect = document.getElementById("startLocation");
const endLocationSelect = document.getElementById("endLocation");
const submitRouteJobButton = document.getElementById(
    "submitRouteJob"
);


// ------------------------------
// Image processing
// ------------------------------

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
            imageUploadID.value = "";

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


// Update compression options when output format changes.
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


// Update the visibility of compression options based on the
// selected output format and original image format.
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

    if (
        originalIsPng &&
        !convertFormatCheckbox.checked
    ) {
        compressionOptions.hidden = true;
        return;
    }

    if (convertingToPng) {
        compressionOptions.hidden = true;
        return;
    }

    compressionOptions.hidden = false;
}


// ------------------------------
// Route processing
// ------------------------------

// Upload the route and distance CSV files.
if (
    routeFileInput &&
    distanceFileInput &&
    routeOptions
) {
    const updateRouteUpload = async () => {
        const routeFile = routeFileInput.files[0];
        const distanceFile = distanceFileInput.files[0];

        routeUploadID.value = "";
        routeOptions.hidden = true;

        if (
            !routeFile ||
            !distanceFile
        ) {
            updateRouteSubmitButton();
            return;
        }

        if (
            routeFile.type &&
            routeFile.type !== "text/csv" &&
            !routeFile.name.toLowerCase().endsWith(".csv")
        ) {
            alert("The route file must be a CSV file.");
            routeFileInput.value = "";
            updateRouteSubmitButton();
            return;
        }

        if (
            distanceFile.type &&
            distanceFile.type !== "text/csv" &&
            !distanceFile.name.toLowerCase().endsWith(".csv")
        ) {
            alert(
                "The distance table file must be a CSV file."
            );
            distanceFileInput.value = "";
            updateRouteSubmitButton();
            return;
        }

        try {
            const formData = new FormData();

            formData.append("routeFile", routeFile);
            formData.append("distanceFile", distanceFile);

            const response = await fetch("/upload/route", {
                method: "POST",
                body: formData
            });

            if (!response.ok) {
                const message = await response.text();
                throw new Error(message);
            }

            const data = await response.json();

            routeUploadID.value = data.upload_id;

            await populateRouteLocations(routeFile);

            routeOptions.hidden = false;
            updateRouteSubmitButton();
        } catch (error) {
            routeUploadID.value = "";
            routeOptions.hidden = true;

            alert(
                `Error uploading route files: ${error.message}`
            );

            routeFileInput.value = "";
            distanceFileInput.value = "";
        }
    };

    routeFileInput.addEventListener(
        "change",
        updateRouteUpload
    );

    distanceFileInput.addEventListener(
        "change",
        updateRouteUpload
    );
}


// Read the route CSV and populate the start/end selectors.
async function populateRouteLocations(file) {
    if (
        !startLocationSelect ||
        !endLocationSelect
    ) {
        return;
    }

    const text = await file.text();

    const rows = parseCSV(text);

    if (rows.length < 2) {
        throw new Error(
            "The route CSV contains no locations."
        );
    }

    const header = rows[0];

    const nameIndex = header.findIndex(
        (column) => column.trim() === "name"
    );

    if (nameIndex === -1) {
        throw new Error(
            "The route CSV is missing the required 'name' column."
        );
    }

    const locationNames = [];

    for (let index = 1; index < rows.length; index++) {
        const row = rows[index];

        if (!row.length) {
            continue;
        }

        const name = row[nameIndex]?.trim();

        if (name && !locationNames.includes(name)) {
            locationNames.push(name);
        }
    }

    if (locationNames.length === 0) {
        throw new Error(
            "The route CSV contains no locations."
        );
    }

    startLocationSelect.replaceChildren();

    const startPlaceholder =
        document.createElement("option");

    startPlaceholder.value = "";
    startPlaceholder.textContent =
        "Select a starting location";

    startLocationSelect.appendChild(
        startPlaceholder
    );

    endLocationSelect.replaceChildren();

    const endDefault =
        document.createElement("option");

    endDefault.value = "";
    endDefault.textContent =
        "Same as starting location";

    endLocationSelect.appendChild(
        endDefault
    );

    for (const name of locationNames) {
        const startOption =
            document.createElement("option");

        startOption.value = name;
        startOption.textContent = name;

        startLocationSelect.appendChild(
            startOption
        );

        const endOption =
            document.createElement("option");

        endOption.value = name;
        endOption.textContent = name;

        endLocationSelect.appendChild(
            endOption
        );
    }
}


// Basic CSV parser that handles quoted fields.
function parseCSV(text) {
    const rows = [];
    let row = [];
    let field = "";
    let insideQuotes = false;

    for (let index = 0; index < text.length; index++) {
        const character = text[index];

        if (character === '"') {
            if (
                insideQuotes &&
                text[index + 1] === '"'
            ) {
                field += '"';
                index++;
            } else {
                insideQuotes = !insideQuotes;
            }

            continue;
        }

        if (character === "," && !insideQuotes) {
            row.push(field);
            field = "";
            continue;
        }

        if (
            (character === "\n" ||
                character === "\r") &&
            !insideQuotes
        ) {
            if (
                character === "\r" &&
                text[index + 1] === "\n"
            ) {
                index++;
            }

            row.push(field);
            field = "";

            if (row.some(
                (value) => value.trim() !== ""
            )) {
                rows.push(row);
            }

            row = [];
            continue;
        }

        field += character;
    }

    if (field !== "" || row.length > 0) {
        row.push(field);

        if (row.some(
            (value) => value.trim() !== ""
        )) {
            rows.push(row);
        }
    }

    return rows;
}


// Update route submission button state.
function updateRouteSubmitButton() {
    if (!submitRouteJobButton) {
        return;
    }

    const ready =
        routeUploadID &&
        routeUploadID.value !== "" &&
        startLocationSelect &&
        startLocationSelect.value !== "";

    submitRouteJobButton.disabled = !ready;
}


// Enable or disable the route submit button when
// the starting location changes.
if (startLocationSelect) {
    startLocationSelect.addEventListener(
        "change",
        updateRouteSubmitButton
    );
}


// ------------------------------
// Initial state
// ------------------------------

updateModelOptions();
updateFeatureOptions();

if (imageOptions) {
    imageOptions.hidden =
        !imageFileInput ||
        imageFileInput.files.length === 0;
}

if (resizeOptions) {
    resizeOptions.hidden =
        !resizeImageCheckbox ||
        !resizeImageCheckbox.checked;
}

if (formatOptions) {
    formatOptions.hidden =
        !convertFormatCheckbox ||
        !convertFormatCheckbox.checked;
}

updateImageCompressionOptions();

if (routeOptions) {
    routeOptions.hidden = true;
}

if (submitRouteJobButton) {
    submitRouteJobButton.disabled = true;
}
