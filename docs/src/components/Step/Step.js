import React from 'react';
import styles from './styles.module.css';
import Highlight from '../Highlight/Highlight';

export default class Step extends React.Component {
  render() {
    let {children, ...props} = this.props

    props.className += ` ${styles.step}`

    return <Highlight {...props}>STEP {children}</Highlight>;
  }
}