use clap::{App, Arg};
mod strtok;
use strtok::strtok;

fn main() {
    let matches = App::new("BCDA API")
        .version("Beta")
        .author("ACO API")
        .arg(
            Arg::new("Configuration")
                .short('c')
                .long("config")
                .value_name("cfg")
                .about("Give me some configuration!!!")
                .takes_value(true),
        )
        .get_matches();

    if let Some(mut i) = matches.value_of("Configuration") {
        let j = strtok(&mut i, ' ');

        if i == "" {
            println!("You have entered {}.", j);
        } else {
            println!("You entered {} first.", j);
            println!("You entered {} second.", i);
        }
    }
}
