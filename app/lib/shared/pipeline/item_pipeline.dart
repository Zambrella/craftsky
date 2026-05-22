import 'dart:async';

/// Function run for one ordered pipeline step.
typedef PipelineStepFunction<T> =
    Future<T> Function(
      T item,
      PipelineStepContext<T> context,
    );

/// One named step in an ordered per-item pipeline.
final class PipelineStep<T> {
  const PipelineStep({required this.name, required this.run});

  /// Stable step name emitted in pipeline events.
  final String name;

  /// Step implementation. It receives the current item value and returns the
  /// next item value for subsequent steps.
  final PipelineStepFunction<T> run;
}

/// Context passed to a step so it can emit progress without mutating state.
final class PipelineStepContext<T> {
  const PipelineStepContext._({
    required this.originalItem,
    required this.stepName,
    required this._emit,
  });

  /// The original item before any pipeline step changed it.
  final T originalItem;

  /// Name of the currently running step.
  final String stepName;

  final void Function(PipelineEvent<T>) _emit;

  /// Emits step-specific progress as a normal pipeline event.
  void reportProgress(Object progress) {
    _emit(StepProgress<T>(originalItem, stepName, progress));
  }
}

/// Event emitted by a pipeline item as it moves through ordered steps.
sealed class PipelineEvent<T> {
  const PipelineEvent(this.item);

  /// Original item for this pipeline run.
  final T item;
}

/// A step has started for an item.
final class StepStarted<T> extends PipelineEvent<T> {
  const StepStarted(super.item, this.stepName);

  /// Step that just started.
  final String stepName;
}

/// A step emitted progress for an item.
final class StepProgress<T> extends PipelineEvent<T> {
  const StepProgress(super.item, this.stepName, this.progress);

  /// Step that emitted progress.
  final String stepName;

  /// Step-defined progress payload.
  final Object progress;
}

/// A step completed successfully for an item.
final class StepCompleted<T> extends PipelineEvent<T> {
  const StepCompleted(super.item, this.stepName, this.result);

  /// Step that completed.
  final String stepName;

  /// Current item after this step completed.
  final T result;
}

/// A step failed for an item. Other items may continue processing.
final class StepFailed<T> extends PipelineEvent<T> {
  const StepFailed(super.item, this.stepName, this.error, this.stackTrace);

  /// Step that failed.
  final String stepName;

  /// Failure thrown by the step.
  final Object error;

  /// Stack trace captured at failure.
  final StackTrace stackTrace;
}

/// All steps completed for an item.
final class ItemCompleted<T> extends PipelineEvent<T> {
  const ItemCompleted(super.item, this.result);

  /// Final item result after the last step.
  final T result;
}

/// Runs ordered steps for each item, with multiple items processed in parallel.
Stream<PipelineEvent<T>> runPipeline<T>({
  required Iterable<T> items,
  required List<PipelineStep<T>> steps,
  int concurrency = 4,
}) {
  final controller = StreamController<PipelineEvent<T>>();
  final itemIterator = items.iterator;
  final workerCount = concurrency < 1 ? 1 : concurrency;

  void emit(PipelineEvent<T> event) {
    if (!controller.isClosed) controller.add(event);
  }

  Future<void> processOne(T originalItem) async {
    var current = originalItem;
    for (final step in steps) {
      emit(StepStarted<T>(originalItem, step.name));
      final context = PipelineStepContext<T>._(
        originalItem: originalItem,
        stepName: step.name,
        emit: emit,
      );
      try {
        current = await Future.sync(() => step.run(current, context));
        emit(StepCompleted<T>(originalItem, step.name, current));
      } on Object catch (error, stackTrace) {
        emit(StepFailed<T>(originalItem, step.name, error, stackTrace));
        return;
      }
    }
    emit(ItemCompleted<T>(originalItem, current));
  }

  Future<void> worker() async {
    while (true) {
      final T item;
      if (itemIterator.moveNext()) {
        item = itemIterator.current;
      } else {
        return;
      }
      await processOne(item);
    }
  }

  unawaited(
    () async {
      try {
        await Future.wait(List.generate(workerCount, (_) => worker()));
      } finally {
        await controller.close();
      }
    }(),
  );

  return controller.stream;
}
