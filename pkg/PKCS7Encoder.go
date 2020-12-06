package pkg

// pkcs7 填充及解密方法从 java sdk 翻译而来

const blockSize = 32

/**
 * 提供基于PKCS7算法的加解密接口.
 */

/**
 * 获得对明文进行补位填充的字节.
 *
 * @param count 需要进行填充补位操作的明文字节个数
 * @return 补齐用的字节数组
 */
func pkcs7Encode(count int) []byte {
	// 计算需要填充的位数
	amountToPad := blockSize - (count % blockSize)
	if amountToPad == 0 {
		amountToPad = blockSize
	}
	// 获得补位所用的字符
	padChr := chr(amountToPad)
	var tmp []byte
	for index := 0; index < amountToPad; index++ {
		tmp = append(tmp, padChr)
	}
	return tmp
}

/**
 * 删除解密后明文的补位字符
 *
 * @param decrypted 解密后的明文
 * @return 删除补位字符后的明文
 */
func pkcs7Decode(decrypted []byte) []byte {
	pad := int(decrypted[len(decrypted)-1])
	if pad < 1 || pad > 32 {
		pad = 0
	}
	return decrypted[:len(decrypted)-pad]
}

/**
 * 将数字转化成ASCII码对应的字符，用于对明文进行补码
 *
 * @param a 需要转化的数字
 * @return 转化得到的字符
 */
func chr(a int) byte {
	return byte(a & 0xFF)
}
