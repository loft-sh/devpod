import React from 'react';
import styles from './styles.module.css';

export default class Highlight extends React.Component {
  render() {
    let {children, ...highlightStyle} = this.props

    return <span style={highlightStyle} className={`${styles.highlight} ${this.props.className}`}> {children} </span>;
  }
}