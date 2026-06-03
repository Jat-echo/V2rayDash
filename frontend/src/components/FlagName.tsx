import { useState } from 'react'
import { getCountryFlag, emojiToSvgUrl } from '../utils/format'

interface FlagNameProps {
  name: string
  style?: React.CSSProperties
}

export function FlagName({ name, style }: FlagNameProps) {
  const emoji = getCountryFlag(name)
  const [imgOk, setImgOk] = useState(true)

  return (
    <span style={style}>
      {imgOk ? (
        <img
          src={emojiToSvgUrl(emoji)}
          alt={emoji}
          onError={() => setImgOk(false)}
          style={{ height: '1em', width: 'auto', verticalAlign: '-0.12em', marginRight: '0.25em', display: 'inline-block' }}
        />
      ) : (
        <span style={{ marginRight: '0.2em' }}>{emoji}</span>
      )}
      {name}
    </span>
  )
}
