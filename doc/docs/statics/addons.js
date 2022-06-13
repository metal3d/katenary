function addSmileLogo() {
  let logo = document.createElement("img");
  logo.src = "/statics/Logo_Smile.png";
  logo.classList.add("smile-logo");
  logo.alt = "Smile logo";

  let link = document.createElement("a");
  link.href = "https://www.smile.eu";
  link.target = "_blank";
  link.title = "Smile website";

  link.appendChild(logo);

  logo.addEventListener("load", () => {
    let side = document.querySelector(".wy-menu");
    side.appendChild(link);
  });
}

function addKatenaryLogo() {
  let logo = document.createElement("img");
  logo.src = "/statics/logo.png";
  logo.classList.add("logo");
  logo.alt = "Katenary logo";

  let link = document.createElement("a");
  link.href = "/";
  link.title = "Index page";

  link.appendChild(logo);

  logo.addEventListener("load", () => {
    let side = document.querySelector(".wy-nav-side");
    side.insertBefore(link, side.firstChild);
  });
}

document.addEventListener("DOMContentLoaded", () => {
  //addKatenaryLogo();
  //addSmileLogo();
});
