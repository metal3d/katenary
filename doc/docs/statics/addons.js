// Install the highlight.js in the documentation. Then
// highlight all the source code.
function hljsInstall() {
  const version = "11.9.0";
  const theme = "github-dark";

  const script = document.createElement("script");
  script.src = `//cdnjs.cloudflare.com/ajax/libs/highlight.js/${version}/highlight.min.js`;
  script.onload = () => {
    const style = document.createElement("link");
    style.rel = "stylesheet";
    style.href = `//cdnjs.cloudflare.com/ajax/libs/highlight.js/${version}/styles/${theme}.min.css`;
    document.head.appendChild(style);
    hljs.highlightAll();
  };

  document.head.appendChild(script);
}

// All images in an .zoomable div is zoomable, that
// meanse that we can click to zoom and unzoom.
// This needs specific CSS (see main.css).
function makeImagesZoomable() {
  const zone = document.querySelectorAll(".zoomable");

  zone.forEach((z, i) => {
    const im = z.querySelectorAll("img");
    if (im.length == 0) {
      return;
    }

    const input = document.createElement("input");
    input.setAttribute("type", "checkbox");
    input.setAttribute("id", `image-zoom-${i}`);
    z.appendChild(input);

    const label = document.createElement("label");
    label.setAttribute("for", `image-zoom-${i}`);
    z.appendChild(label);

    label.appendChild(im[0]);
  });
}

document.addEventListener("DOMContentLoaded", () => {
  hljsInstall();
  makeImagesZoomable();
});
