"""NF command/write persistence helpers."""

from unittest.mock import MagicMock

import pandas as pd
import pytest

from openfdd_nf.config import AppConfig, FaultOutputTarget, FetchConfig, NFConfig, RunConfig
from openfdd_nf.nf_persist import persist_fault_flags, should_persist


def test_should_persist_respects_mapping_and_flags():
    cfg = AppConfig(
        nf=NFConfig(base_url="http://x"),
        fetch=FetchConfig(),
        run=RunConfig(rules_dir="/tmp"),
        point_uuids={"X": "u"},
        fault_outputs={"f1": FaultOutputTarget(uuid="u1", layer="L")},
    )
    assert should_persist(cfg, None, False) is False
    assert should_persist(cfg, True, False) is True
    assert should_persist(cfg, False, True) is False


def test_persist_fault_flags_posts_writes():
    df = pd.DataFrame(
        {
            "f1": [0, 0, 1],
        }
    )
    client = MagicMock()
    client.post.return_value.status_code = 200
    client.post.return_value.json.return_value = {"results": [], "errors": []}
    targets = {"f1": FaultOutputTarget(uuid="uuid-1", layer="hpl:bacnet:1")}
    out = persist_fault_flags(client, df, targets)
    assert out["ok"] is True
    client.post.assert_called_once()
    call_kw = client.post.call_args[1]
    assert "json" in call_kw
    writes = call_kw["json"]["writes"]
    assert len(writes) == 1
    assert writes[0]["value"]["real"] == 1.0
