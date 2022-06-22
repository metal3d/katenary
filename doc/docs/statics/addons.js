function hljsInstall() {
  const version = "11.5.1";
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

document.addEventListener("DOMContentLoaded", () => {
  hljsInstall();
});
