using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using CommandLine;
using Org.BouncyCastle.Crypto;
using Org.BouncyCastle.Crypto.Digests;
using Org.BouncyCastle.Crypto.Encodings;
using Org.BouncyCastle.Crypto.Engines;
using Org.BouncyCastle.Crypto.Modes;
using Org.BouncyCastle.Crypto.Parameters;
using Org.BouncyCastle.OpenSsl;

// Code based on https://www.codeproject.com/Articles/1265115/Cross-Platform-AES-256-GCM-Encryption-Decryption
//           and https://gist.github.com/dziwoki/cc41b523c2bd43ee646b957f0aa91943

namespace BCDA
{
    class Decrypt
    {
        public static readonly int NonceByteSize = 12;

        public class Options
        {
            [Option('k', "key",
                HelpText = "Encrypted symmetric key used for file decryption (hex encoded string).",
                Required = true)]
            public string Key { get; set; }

            [Option('f', "file",
                HelpText = "Location of encrypted file.",
                Required = true)]
            public string File { get; set; }

            [Option('p', "pk",
                HelpText = "Location of private key to use for decryption of symmetric key.",
                Required = true)]
            public string PrivateKey { get; set; }
        }

        static void Main(string[] args)
        {
            Parser.Default.ParseArguments<Options>(args)
               .WithParsed<Options>(o =>
               {
                   string ndjson = PerformDecryption(o.File, o.PrivateKey, o.Key);
                   Console.WriteLine(ndjson);
               });
        }

        private static string PerformDecryption(string encryptedFilePath, string privateKeyPath, string encSymmetricKey)
        {
            // Use the encrypted file's name as the label in decrypting the private key.
            // Note that this means you MUST save it with the filename given by the API.
            string label = new FileInfo(encryptedFilePath).Name;
            byte[] symmetricKey = DecodeSymmetricKey(label, privateKeyPath, encSymmetricKey);

            byte[] fileText = File.ReadAllBytes(encryptedFilePath);
            byte[] nonce = new ArraySegment<byte>(fileText, 0, NonceByteSize).ToArray();
            byte[] encryptedText = new ArraySegment<byte>(fileText, NonceByteSize, fileText.Length - NonceByteSize).ToArray();

            return DecryptFile(encryptedText, symmetricKey, nonce);
        }

        private static byte[] DecodeSymmetricKey(string label, string privateKeyPath, string ciphertext)
        {
            byte[] cipherTextBytes = HexToByte(ciphertext);
            byte[] labelBytes = Encoding.UTF8.GetBytes(label);

            PemReader pr = new PemReader(File.OpenText(privateKeyPath));
            AsymmetricCipherKeyPair keys = (AsymmetricCipherKeyPair)pr.ReadObject();

            OaepEncoding eng = new OaepEncoding(new RsaEngine(), new Sha256Digest(), new Sha256Digest(), labelBytes);
            eng.Init(false, keys.Private);

            int length = cipherTextBytes.Length;
            int blockSize = eng.GetInputBlockSize();
            List<byte> plainTextBytes = new List<byte>();
            for (int chunkPosition = 0;
                chunkPosition < length;
                chunkPosition += blockSize)
            {
                int chunkSize = Math.Min(blockSize, length - chunkPosition);
                plainTextBytes.AddRange(eng.ProcessBlock(
                    cipherTextBytes, chunkPosition, chunkSize
                ));
            }
            return plainTextBytes.ToArray();
        }

        private static string DecryptFile(byte[] encryptedText, byte[] key, byte[] nonce)
        {
            string plaintext = string.Empty;
            try
            {
                GcmBlockCipher cipher = new GcmBlockCipher(new AesEngine());
                AeadParameters parameters = new AeadParameters(new KeyParameter(key), 128, nonce);

                cipher.Init(false, parameters);
                byte[] plainBytes = new byte[cipher.GetOutputSize(encryptedText.Length)];
                Int32 byteLen = cipher.ProcessBytes(encryptedText, 0, encryptedText.Length, plainBytes, 0);
                cipher.DoFinal(plainBytes, byteLen);

                plaintext = Encoding.UTF8.GetString(plainBytes);
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
                Console.WriteLine(ex.StackTrace);
            }

            return plaintext;
        }

        private static Byte[] HexToByte(string hexStr)
        {
            byte[] bArray = new byte[hexStr.Length / 2];
            for (int i = 0; i < (hexStr.Length / 2); i++)
            {
                byte firstNibble = Byte.Parse(hexStr.Substring((2 * i), 1), System.Globalization.NumberStyles.HexNumber);
                byte secondNibble = Byte.Parse(hexStr.Substring((2 * i) + 1, 1), System.Globalization.NumberStyles.HexNumber);
                int finalByte = (secondNibble) | (firstNibble << 4);
                bArray[i] = (byte)finalByte;
            }
            return bArray;
        }
    }
}