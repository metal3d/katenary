"""Called by mkdocs to preprocess files and build manpages"""

from bs4 import BeautifulSoup, Tag


def to_remove(tag: Tag) -> bool:
    """Removes images, SVGs, links containing images or SVGs, and permalinks from the BeautifulSoup object."""
    if tag.name in {"img", "svg"}:
        return True
    # remove links containing images or SVGs
    if tag.name == "a" and tag.img and to_remove(tag.img):
        return True
    # remove permalinks
    if tag.name == "a" and "headerlink" in tag.get("class", ()):
        return True
    return False


def preprocess(soup: BeautifulSoup, output: str) -> None:
    """Preprocess the BeautifulSoup object to remove unwanted elements."""
    for element in soup.find_all(to_remove):
        element.decompose()
