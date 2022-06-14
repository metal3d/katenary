function addSmileLogo() {
  const logo = document.createElement("img");
  logo.src = "/statics/Logo_Smile.png";
  logo.classList.add("smile-logo");
  logo.alt = "Smile logo";

  const link = document.createElement("a");
  link.href = "https://www.smile.eu";
  link.target = "_blank";
  link.title = "Smile website";
  link.classList.add("smile-logo");
  link.appendChild(logo);

  const text = document.createElement("p");
  text.innerHTML = "Sponsored by Smile";

  const div = document.createElement("div");
  div.classList.add("smile-logo");
  div.appendChild(text);
  div.appendChild(link);

  logo.addEventListener("load", () => {
    let side = document.querySelector(".md-footer-meta__inner");
    side.appendChild(div);
  });
}

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
    hljs.initHighlightingOnLoad();
  };

  document.head.appendChild(script);
}

document.addEventListener("DOMContentLoaded", () => {
  addSmileLogo();
  hljsInstall();
});
