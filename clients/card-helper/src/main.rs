mod reader;

use pcsc::*;

fn main() {
    // Establish a PC/SC context.
    let ctx = match Context::establish(Scope::User) {
        Ok(ctx) => ctx,
        Err(err) => {
            eprintln!("Failed to establish context: {}", err);
            std::process::exit(1);
        }
    };

    // List available readers.
    let mut readers_buf = [0; 2048];
    let mut readers = match ctx.list_readers(&mut readers_buf) {
        Ok(readers) => readers,
        Err(err) => {
            eprintln!("Failed to list readers: {}", err);
            std::process::exit(1);
        }
    };

    // Use the first reader.
    let reader = match readers.next() {
        Some(reader) => reader,
        None => {
            println!("No readers are connected.");
            return;
        }
    };
    println!("Using reader: {:?}", reader);

    // Connect to the card.
    let card = match ctx.connect(reader, ShareMode::Shared, Protocols::ANY) {
        Ok(card) => card,
        Err(Error::NoSmartcard) => {
            println!("A smartcard is not present in the reader.");
            return;
        }
        Err(err) => {
            eprintln!("Failed to connect to card: {}", err);
            std::process::exit(1);
        }
    };

    // Send an APDU command.
    let apdu = b"\xFF\xCA\x00\x00\x00";
    println!("Sending APDU: {:?}", apdu);
    let mut rapdu_buf = [0; MAX_BUFFER_SIZE];
    let rapdu = match card.transmit(apdu, &mut rapdu_buf) {
        Ok(rapdu) => rapdu,
        Err(err) => {
            eprintln!("Failed to transmit APDU command to card: {}", err);
            std::process::exit(1);
        }
    };
    let rapdu = &rapdu[..rapdu.len() - 2];
    // Convert the response APDU to a hex string.
    let rapdu_hex = rapdu.iter().map(|b| format!("{:02X}", b)).collect::<String>();
    println!("Received APDU: {}", rapdu_hex);
}
