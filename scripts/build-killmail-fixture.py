#!/usr/bin/env python3
"""Build the static-data fixture the killmail corpus tests parse against.

The corpus is a hundred-odd real ESI killmails. Parsing one needs the SDE — the
ship's group, the group's category, a module's meta level — and the Jita price of
every type on the mail as of the day it died. Loading the whole SDE for that
would mean a database in CI; loading only the rows the corpus actually touches
means a file.

Reads the corpus from internal/killmail/testdata/corpus/ and a database holding
the SDE and price history (the local stack), and writes
internal/killmail/testdata/sde.json.

    python3 scripts/build-killmail-fixture.py

Re-run it when the corpus changes. The output is committed, so the tests need
neither a database nor a network.
"""
import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CORPUS = ROOT / "internal/killmail/testdata/corpus"
OUT = ROOT / "internal/killmail/testdata/sde.json"

DSN = os.environ.get(
    "DATABASE_URL",
    "postgresql://evekill:" "evekill@127.0.0.1:5432/evekill",
)
CONTAINER = os.environ.get("PG_CONTAINER", "evekill-go-postgres-1")

# Only these dogma attributes are ever read by a killmail computation:
# 1547 rigSize, 633 metaLevel, 1211 heatDamage.
DOGMA_ATTRIBUTES = (1547, 633, 1211)


def psql(sql):
    """Run a query and return rows as lists of strings."""
    out = subprocess.run(
        ["docker", "exec", "-i", CONTAINER, "psql", "-U", "evekill", "-d", "evekill", "-At", "-F", "\x1f", "-c", sql],
        capture_output=True, text=True,
    )
    if out.returncode != 0:
        sys.exit(f"psql failed: {out.stderr.strip()}")
    return [line.split("\x1f") for line in out.stdout.splitlines() if line]


def collect_items(items, into):
    """Walk the two-level ESI item tree, collecting every type id."""
    for item in items or []:
        into.add(int(item["item_type_id"]))
        collect_items(item.get("items"), into)


def main():
    killmails = sorted(CORPUS.glob("*.json"))
    if not killmails:
        sys.exit(f"no corpus files in {CORPUS}")

    types, systems = set(), set()
    # date -> the types whose price that day's kills need.
    per_date = {}

    for path in killmails:
        km = json.loads(path.read_text())
        date = km["killmail_time"][:10]
        needed = per_date.setdefault(date, set())

        systems.add(int(km["solar_system_id"]))

        victim = km.get("victim", {})
        if victim.get("ship_type_id"):
            types.add(int(victim["ship_type_id"]))
            needed.add(int(victim["ship_type_id"]))

        on_mail = set()
        collect_items(victim.get("items"), on_mail)
        types |= on_mail
        needed |= on_mail

        for att in km.get("attackers", []):
            for key in ("ship_type_id", "weapon_type_id"):
                if att.get(key):
                    types.add(int(att[key]))

    print(f"{len(killmails)} killmails, {len(types)} types, {len(systems)} systems, {len(per_date)} dates")

    def id_list(ids):
        return ",".join(str(i) for i in sorted(ids)) or "-1"

    fixture = {}

    fixture["types"] = {
        r[0]: {
            "group_id": int(r[1]), "category_id": int(r[2]), "name": r[3],
            "variation_parent_type_id": int(r[4]), "meta_group_id": int(r[5]),
            "market_group_id": int(r[6]),
        }
        for r in psql(f"""
            SELECT type_id, coalesce(group_id,0), coalesce(category_id,0), coalesce(name,''),
                   coalesce(variation_parent_type_id,0), coalesce(meta_group_id,0),
                   coalesce(market_group_id,0)
            FROM inv_types WHERE type_id IN ({id_list(types)})""")
    }

    group_ids = {t["group_id"] for t in fixture["types"].values() if t["group_id"]}
    fixture["groups"] = {
        r[0]: {"category_id": int(r[1]), "name": r[2]}
        for r in psql(f"""
            SELECT group_id, coalesce(category_id,0), coalesce(name,'')
            FROM inv_groups WHERE group_id IN ({id_list(group_ids)})""")
    }

    fixture["systems"] = {
        r[0]: {
            "constellation_id": int(r[1]), "region_id": int(r[2]),
            "name": r[3], "security": float(r[4]),
        }
        for r in psql(f"""
            SELECT solar_system_id, coalesce(constellation_id,0), coalesce(region_id,0),
                   coalesce(system_name,''), coalesce(security,0)
            FROM solar_systems WHERE solar_system_id IN ({id_list(systems)})""")
    }

    region_ids = {s["region_id"] for s in fixture["systems"].values() if s["region_id"]}
    constellation_ids = {s["constellation_id"] for s in fixture["systems"].values() if s["constellation_id"]}

    fixture["regions"] = {
        r[0]: {"name": r[1]}
        for r in psql(f"SELECT region_id, coalesce(name,'') FROM regions WHERE region_id IN ({id_list(region_ids)})")
    }
    fixture["constellations"] = {
        r[0]: {"name": r[1], "region_id": int(r[2])}
        for r in psql(f"""
            SELECT constellation_id, coalesce(constellation_name,''), coalesce(region_id,0)
            FROM constellations WHERE constellation_id IN ({id_list(constellation_ids)})""")
    }

    # Keyed "typeID:attributeID" so the whole thing stays a flat JSON object.
    fixture["dogma"] = {
        f"{r[0]}:{r[1]}": float(r[2])
        for r in psql(f"""
            SELECT type_id, attribute_id, value FROM type_dogma_attributes
            WHERE type_id IN ({id_list(types)})
              AND attribute_id IN ({",".join(str(a) for a in DOGMA_ATTRIBUTES)})""")
    }

    # Small enough to take whole, and the overrides apply to every date.
    fixture["custom_prices"] = {
        r[0]: float(r[1])
        for r in psql("SELECT DISTINCT ON (type_id) type_id, price FROM custom_prices ORDER BY type_id, date DESC")
    }

    # One price per type per kill date: the most recent average at or before that
    # day, which is exactly what the parser resolves.
    prices = {}
    for date, needed in sorted(per_date.items()):
        rows = psql(f"""
            SELECT DISTINCT ON (type_id) type_id, average FROM prices
            WHERE type_id IN ({id_list(needed)}) AND date <= '{date}'::date AND average > 0
            ORDER BY type_id, date DESC""")
        prices[date] = {r[0]: float(r[1]) for r in rows}
    fixture["prices"] = prices

    OUT.write_text(json.dumps(fixture, indent=1, sort_keys=True) + "\n")
    size = OUT.stat().st_size
    print(f"wrote {OUT.relative_to(ROOT)} ({size / 1024:.0f} KiB)")
    print(f"  types={len(fixture['types'])} groups={len(fixture['groups'])} "
          f"systems={len(fixture['systems'])} dogma={len(fixture['dogma'])} "
          f"price-days={len(prices)} price-entries={sum(len(v) for v in prices.values())}")


main()
