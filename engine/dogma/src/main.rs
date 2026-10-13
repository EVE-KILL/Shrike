use esf_data::Sde;
use serde::Deserialize;
use serde_json::{Value, json};
use shrike_dogma::{LegacyFit, attribute_id, calculate};
use std::{
    collections::BTreeMap,
    io::{self, BufRead, Write},
    path::PathBuf,
};
mod fields;
#[derive(Deserialize)]
struct Request {
    id: i64,
    fit: LegacyFit,
    skills: Option<BTreeMap<i32, u8>>,
}
fn main() -> Result<(), Box<dyn std::error::Error>> {
    let path = PathBuf::from(
        std::env::args()
            .nth(1)
            .ok_or("usage: shrike-dogma <SDE directory>")?,
    );
    let bytes = std::fs::read(if path.is_dir() {
        path.join("sde.dat")
    } else {
        path
    })?;
    let data = Sde::new(&bytes)?;
    let fields: Vec<_> = fields::FIELDS
        .iter()
        .map(|(key, name, resist)| (*key, attribute_id(&data, name), *resist))
        .collect();
    let mut output = io::stdout().lock();
    for line in io::stdin().lock().lines() {
        let line = line?;
        let id = serde_json::from_str::<Value>(&line)
            .ok()
            .and_then(|v| v.get("id").and_then(Value::as_i64))
            .unwrap_or(0);
        let response = match serde_json::from_str::<Request>(&line) {
            Ok(req) => {
                std::panic::catch_unwind(|| match calculate(&data, req.fit, req.skills, true) {
                    Ok(stats) => {
                        let hull: serde_json::Map<_, _> = fields
                            .iter()
                            .map(|(key, id, resist)| {
                                let value = id
                                    .and_then(|id| stats.hull.attributes.get(&id))
                                    .and_then(|a| a.value);
                                (
                                    (*key).into(),
                                    value
                                        .map(|v| json!(if *resist { 1.0 - v } else { v }))
                                        .unwrap_or(Value::Null),
                                )
                            })
                            .collect();
                        json!({"id":req.id,"hull":hull})
                    }
                    Err(error) => json!({"id":req.id,"error":error}),
                })
                .unwrap_or_else(|_| json!({"id":id,"error":"dogma calculation failed"}))
            }
            Err(error) => json!({"id":id,"error":error.to_string()}),
        };
        writeln!(output, "{}", response)?;
        output.flush()?;
    }
    Ok(())
}
