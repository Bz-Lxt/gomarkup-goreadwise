"""API smoke in mock mode. Expected cost: ¥0."""
from __future__ import annotations

import json
import os
import uuid
import urllib.error
import urllib.request

BASE = os.environ.get("API_BASE", "http://127.0.0.1:15232/api/v1")


def req(method: str, path: str, body=None, expect=200):
    data = None if body is None else json.dumps(body).encode()
    r = urllib.request.Request(BASE + path, data=data, method=method)
    r.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(r, timeout=15) as resp:
            raw = json.loads(resp.read().decode())
            assert resp.status == expect, resp.status
            return raw
    except urllib.error.HTTPError as e:
        raw = json.loads(e.read().decode())
        if e.code != expect:
            raise AssertionError(f"{method} {path} -> {e.code} {raw}") from e
        return raw


def test_health():
    d = req("GET", "/health")
    assert d["data"]["status"] == "ok"
    assert d["data"]["clip_mode"] == "mock"


def test_list_and_graph():
    cards = req("GET", "/cards")
    assert cards["meta"]["total"] >= 30
    g = req("GET", "/graph")
    assert len(g["data"]["nodes"]) >= 30
    assert len(g["data"]["edges"]) >= 1


def test_wikilink_backlink_immediate():
    title = "SmokeSrc-" + uuid.uuid4().hex[:8]
    src = req(
        "POST",
        "/cards",
        {"title": title, "body": "see [[Zettelkasten]]"},
        expect=201,
    )
    z = None
    for c in req("GET", "/cards?q=Zettelkasten")["data"]:
        if c["title"] == "Zettelkasten":
            z = c
            break
    assert z
    links = req("GET", f"/cards/{z['id']}/links")
    titles = [x.get("source_title") for x in links["data"]["back_links"]]
    assert title in titles
    req("DELETE", f"/cards/{src['data']['id']}")


def test_code_fence_not_linked():
    created = req(
        "POST",
        "/cards",
        {"title": "SmokeFence-" + uuid.uuid4().hex[:8], "body": "```\\n[[NotANode]]\\n```\\n"},
        expect=201,
    )
    detail = req("GET", f"/cards/{created['data']['id']}")
    assert detail["data"]["out_links"] in (None, [])
    req("DELETE", f"/cards/{created['data']['id']}")


def test_clip_mock():
    try:
        card = req("POST", "/clips", {"url": "https://1.1.1.1/fixture-default"}, expect=201)
        assert card["data"]["title"]
        req("DELETE", f"/cards/{card['data']['id']}")
    except AssertionError as e:
        # 409: fixture title already exists. 403: sandbox DNS maps public hosts to RFC1918.
        if "409" in str(e) or "403" in str(e) or "CLIP_DENIED" in str(e):
            return
        raise


if __name__ == "__main__":
    test_health()
    test_list_and_graph()
    test_wikilink_backlink_immediate()
    test_code_fence_not_linked()
    test_clip_mock()
    print("api_smoke PASS cost=¥0")
