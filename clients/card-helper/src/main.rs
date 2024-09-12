mod reader;

use pcsc::*;
use reader::AsyncPcsc;

#[tokio::main]
async fn main() {
    let mut ctx = AsyncPcsc::establish().unwrap();
    let readers = ctx.list_readers().await.unwrap();
    println!("Readers: {:?}", readers);

    ctx.set_reader(&readers[0]);

    let card = ctx.connect().await.unwrap();
    println!("Connected to card.");

    let card_id = ctx.get_card_id(&card).await.unwrap();
    println!("Received APDU: {}", card_id);
}
