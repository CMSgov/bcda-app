pub fn strtok<'a, 'b: 'a>(s: &'a mut &'b str, delimiter: char) -> &'b str {
    if let Some(i) = s.find(delimiter) {
        let prefix = &s[..i];
        let suffix = &s[(i + 1)..];
        *s = suffix; // suffix and s has the same lifetime
        prefix
    } else {
        let prefix = *s;
        *s = ""; // The empty string is assigned here so it has same lifetime
        prefix
    }
}

