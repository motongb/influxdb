// Libraries
import React, {PureComponent} from 'react'

// Components
import {ComponentSize, Label, ResourceCard} from '@influxdata/clockface'

// Types
import {ILabel} from '@influxdata/influx'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'
import LabelContextMenu from './LabelContextMenu'

interface Props {
  label: ILabel
  onClick: (labelID: string) => void
  onDelete: (labelID: string) => void
}

@ErrorHandling
export default class LabelCard extends PureComponent<Props> {
  public render() {
    const {label, onDelete} = this.props

    return (
      <>
        <ResourceCard
          testID="label-card"
          contextMenu={<LabelContextMenu label={label} onDelete={onDelete} />}
          name={
            <Label
              id={label.id}
              name={label.name}
              color={label.properties.color}
              description={label.properties.description}
              size={ComponentSize.Small}
              onClick={this.handleClick}
            />
          }
          metaData={[<>Description: {label.properties.description}</>]}
        />
      </>
    )
  }

  private handleClick = (): void => {
    const {label, onClick} = this.props

    onClick(label.id)
  }
}
