use pcsc::*;

use std::{ffi::CString, sync::Arc, time::Duration};
use tokio::sync::Mutex;

pub struct AsyncPcsc {
    ctx: Arc<Mutex<pcsc::Context>>,
    reader: Option<CString>,
    card_wait_timeout: Option<Duration>,
}

impl AsyncPcsc {
    pub fn establish() -> Result<Self, pcsc::Error> {
        let context = pcsc::Context::establish(Scope::User)?;
        Ok(AsyncPcsc {
            ctx: Arc::new(Mutex::new(context)),
            reader: None,
            card_wait_timeout: None,
        })
    }

    pub fn set_card_wait_timeout(&mut self, timeout: Duration) {
        self.card_wait_timeout = Some(timeout);
    }

    pub async fn list_readers(&self) -> Result<Vec<CString>, pcsc::Error> {
        let context = self.ctx.lock().await;
        let readers = context.list_readers_owned()?;
        Ok(readers)
    }

    pub fn set_reader(&mut self, reader: &CString) {
        self.reader = Some(reader.clone());
    }

    pub async fn try_connect(&self) -> Result<pcsc::Card, pcsc::Error> {
        if self.reader.is_none() {
            return Err(pcsc::Error::NoSmartcard);
        }
        let context = self.ctx.lock().await;
        let card = context.connect(self.reader.as_ref().unwrap(), ShareMode::Shared, Protocols::ANY)?;
        Ok(card)
    }

    pub async fn connect(&self) -> Result<pcsc::Card, pcsc::Error> {
        let _thread_ctx = self.ctx.clone();
        loop {
            match self.try_connect().await {
                Ok(card) => return Ok(card),
                Err(pcsc::Error::NoSmartcard) => {
                    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                }
                Err(err) => {
                    return Err(err);
                }
            }
        }
    }

    pub async fn transmit(&self, card: &pcsc::Card, apdu: &[u8]) -> Result<Vec<u8>, pcsc::Error> {
        let _ = self.ctx.lock().await;
        let mut rapdu_buf = [0; MAX_BUFFER_SIZE];
        let rapdu = card.transmit(apdu, &mut rapdu_buf)?;
        Ok(rapdu.to_vec())
    }

    pub async fn get_card_id(&self, card: &pcsc::Card) -> Result<String, pcsc::Error> {
        let apdu = b"\xFF\xCA\x00\x00\x00";
        let rapdu = self.transmit(card, apdu).await?;
        let rapdu = &rapdu[..rapdu.len() - 2];
        let rapdu_hex = rapdu.iter().map(|b| format!("{:02X}", b)).collect::<String>();
        Ok(rapdu_hex)
    }
}